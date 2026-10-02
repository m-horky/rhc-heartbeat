package otlp

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/m-horky/rhc-heartbeat/internal/constants"
	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"go.opentelemetry.io/collector/pdata/plog"
)

const (
	requestTimeout  = 10 * time.Second
	connectTimeout  = 5 * time.Second
	maxResponseBody = 64 * 1024
)

// Client sends heartbeat batches to a configured OTLP/HTTP logs endpoint.
type Client struct {
	endpoint string
	http     *http.Client
}

// New constructs an OTLP client using the resolved application configuration.
func New(cfg config.Config) (*Client, error) {
	endpoint, err := validateEndpoint(cfg.OTEL.URI)
	if err != nil {
		return nil, err
	}

	httpClient, err := newHTTPClient(cfg, endpoint)
	if err != nil {
		return nil, err
	}

	return &Client{endpoint: endpoint, http: httpClient}, nil
}

// Upload encodes and sends the supplied heartbeats as one OTLP/JSON request.
func (client *Client) Upload(ctx context.Context, heartbeats []heartbeat.Heartbeat) error {
	if client == nil || client.http == nil {
		return errors.New("upload OTLP logs: missing client")
	}

	if len(heartbeats) == 0 {
		return nil
	}

	logs := buildLogs(heartbeats, time.Now())

	payload, err := (&plog.JSONMarshaler{}).MarshalLogs(logs)
	if err != nil {
		return fmt.Errorf("marshal OTLP/JSON logs: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create OTLP request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := client.http.Do(request)
	if err != nil {
		return fmt.Errorf("send OTLP request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	responseBody, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBody+1))
	if err != nil {
		return fmt.Errorf("read OTLP response: %w", err)
	}

	if len(responseBody) > maxResponseBody {
		return fmt.Errorf("read OTLP response: body exceeds %d bytes", maxResponseBody)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("OTLP endpoint returned %s: %s", response.Status, strings.TrimSpace(string(responseBody)))
	}

	if len(responseBody) == 0 {
		return nil
	}

	var result exportLogsResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return fmt.Errorf("decode OTLP response: %w", err)
	}

	if result.PartialSuccess != nil &&
		(result.PartialSuccess.RejectedLogRecords > 0 || result.PartialSuccess.ErrorMessage != "") {
		return fmt.Errorf("OTLP endpoint partially rejected logs: rejected=%d message=%q",
			result.PartialSuccess.RejectedLogRecords, result.PartialSuccess.ErrorMessage)
	}

	return nil
}

// CloseIdleConnections closes idle connections held by the client's HTTP transport.
func (client *Client) CloseIdleConnections() {
	if client != nil && client.http != nil {
		client.http.CloseIdleConnections()
	}
}

// validateEndpoint parses and validates a configured OTLP endpoint URI.
func validateEndpoint(endpoint string) (string, error) {
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid OTLP endpoint URL %q", endpoint)
	}

	return endpoint, nil
}

// newHTTPClient creates an HTTP client configured with the endpoint TLS and proxy settings.
func newHTTPClient(cfg config.Config, endpoint string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	// Do not inherit environment proxy
	transport.Proxy = nil
	transport.DialContext = (&net.Dialer{Timeout: connectTimeout}).DialContext
	transport.IdleConnTimeout = 30 * time.Second

	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse OTLP endpoint URL: %w", err)
	}

	if parsedEndpoint.Scheme == "https" {
		tlsConfig, err := newTLSConfig(cfg)
		if err != nil {
			return nil, err
		}

		transport.TLSClientConfig = tlsConfig
	}

	if err := configureProxy(transport, cfg.HTTP.Proxy); err != nil {
		return nil, err
	}

	return &http.Client{
		Transport:     transport,
		Timeout:       requestTimeout,
		CheckRedirect: preventCrossOriginRedirect,
	}, nil
}

// preventCrossOriginRedirect rejects redirects that change the request origin.
func preventCrossOriginRedirect(request *http.Request, via []*http.Request) error {
	if len(via) == 0 {
		return nil
	}

	previous := via[len(via)-1].URL
	if !strings.EqualFold(previous.Scheme, request.URL.Scheme) ||
		!strings.EqualFold(previous.Hostname(), request.URL.Hostname()) ||
		effectivePort(previous) != effectivePort(request.URL) {
		return errors.New("refusing cross-origin redirect")
	}

	return nil
}

// effectivePort returns the URL port or the default port for HTTP(S).
func effectivePort(value *url.URL) string {
	if port := value.Port(); port != "" {
		return port
	}

	switch strings.ToLower(value.Scheme) {
	case "http":
		return "80"
	case "https":
		return "443"
	default:
		return ""
	}
}

// newTLSConfig creates the endpoint TLS configuration, including configured certificate verification.
func newTLSConfig(cfg config.Config) (*tls.Config, error) {
	certificate, err := tls.LoadX509KeyPair(
		constants.PathFromEnv(constants.ClientCertificatePathEnv, constants.DefaultClientCertificatePath),
		constants.PathFromEnv(constants.ClientKeyPathEnv, constants.DefaultClientKeyPath),
	)
	if err != nil {
		return nil, fmt.Errorf("load OTLP client certificate: %w", err)
	}

	rootCAs, err := loadRootCAs(cfg.OTEL.CAPath)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      rootCAs,
		//nolint:gosec // Configuration intentionally controls server certificate validation.
		InsecureSkipVerify: !cfg.OTEL.TLSVerify,
	}, nil
}

// loadRootCAs loads system roots and appends the optionally configured CA certificate.
func loadRootCAs(caPath string) (*x509.CertPool, error) {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("load system CA certificates: %w", err)
	}

	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	if caPath == "" {
		return rootCAs, nil
	}

	caData, err := (fs.Filesystem{}).Read(caPath)
	if err != nil {
		return nil, fmt.Errorf("read OTLP CA certificate %s: %w", caPath, err)
	}

	if !rootCAs.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("read OTLP CA certificate %s: no certificates found", caPath)
	}

	return rootCAs, nil
}

// configureProxy applies the configured proxy URI and credentials to the transport.
func configureProxy(transport *http.Transport, proxy config.Proxy) error {
	if proxy.URI == "" {
		if proxy.User != "" || proxy.Password != "" {
			return errors.New("configure HTTP proxy: credentials require a proxy URI")
		}

		return nil
	}

	proxyURL, err := url.Parse(proxy.URI)
	if err != nil {
		return fmt.Errorf("parse HTTP proxy URL: %w", err)
	}

	if strings.EqualFold(proxyURL.Scheme, "http") &&
		(proxy.User != "" || proxy.Password != "" || proxyURL.User != nil) {
		return errors.New("configure HTTP proxy: credentials require an HTTPS proxy URI")
	}

	if proxyURL.User != nil {
		return errors.New("configure HTTP proxy: credentials must not be embedded in the proxy URI")
	}

	if proxy.User != "" || proxy.Password != "" {
		proxyURL.User = url.UserPassword(proxy.User, proxy.Password)
	}

	transport.Proxy = http.ProxyURL(proxyURL)

	return nil
}

type exportLogsResponse struct {
	PartialSuccess *partialSuccess `json:"partialSuccess"`
}

type partialSuccess struct {
	RejectedLogRecords int64  `json:"rejectedLogRecords,string"`
	ErrorMessage       string `json:"errorMessage"`
}
