package remotewrite

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang/snappy"
	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

const (
	requestTimeout = 10 * time.Second
	connectTimeout = 5 * time.Second
)

// Options contains application-specific values used to configure the Remote Write client.
type Options struct {
	// ClientCertificatePath is the path to the mutual TLS client certificate.
	ClientCertificatePath string
	// ClientKeyPath is the path to the mutual TLS client key.
	ClientKeyPath string
	// UserAgent is sent with each Remote Write request.
	UserAgent string
}

// Client sends heartbeat batches to a Prometheus Remote Write v1 endpoint.
type Client struct {
	endpoint  string
	http      *http.Client
	userAgent string
}

// New constructs a Remote Write client using resolved heartbeat configuration and supplied options.
func New(cfg config.Config, options Options) (*Client, error) {
	endpoint, err := validateEndpoint(cfg.Heartbeat.URI.String())
	if err != nil {
		return nil, err
	}

	httpClient, err := newHTTPClient(cfg, endpoint, options)
	if err != nil {
		return nil, err
	}

	return &Client{endpoint: endpoint, http: httpClient, userAgent: options.UserAgent}, nil
}

// Upload encodes and sends the supplied heartbeats as one Snappy-compressed Remote Write request.
func (client *Client) Upload(ctx context.Context, heartbeats []heartbeat.Heartbeat) error {
	if client == nil || client.http == nil {
		return errors.New("upload Prometheus Remote Write samples: missing client")
	}

	if len(heartbeats) == 0 {
		return nil
	}

	payload, err := encodeHeartbeats(heartbeats)
	if err != nil {
		return err
	}

	compressed := snappy.Encode(nil, payload)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("create Prometheus Remote Write request: %w", err)
	}

	request.Header.Set("Content-Encoding", "snappy")
	request.Header.Set("Content-Type", "application/x-protobuf")
	request.Header.Set("User-Agent", client.userAgent)
	request.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	response, err := client.http.Do(request)
	if err != nil {
		return fmt.Errorf("send Prometheus Remote Write request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("prometheus Remote Write endpoint returned %s", response.Status)
	}

	return nil
}

// CloseIdleConnections closes idle connections held by the client's HTTP transport.
func (client *Client) CloseIdleConnections() {
	if client != nil && client.http != nil {
		client.http.CloseIdleConnections()
	}
}

// validateEndpoint parses and validates a configured Remote Write endpoint URI.
func validateEndpoint(endpoint string) (string, error) {
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("invalid Prometheus Remote Write endpoint URL %q", endpoint)
	}

	return endpoint, nil
}

// newHTTPClient creates an HTTP client configured with endpoint TLS and proxy settings.
func newHTTPClient(cfg config.Config, endpoint string, options Options) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = (&net.Dialer{Timeout: connectTimeout}).DialContext
	transport.IdleConnTimeout = 30 * time.Second

	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse Prometheus Remote Write endpoint URL: %w", err)
	}

	if parsedEndpoint.Scheme == "https" {
		tlsConfig, err := newTLSConfig(cfg, options)
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
func newTLSConfig(cfg config.Config, options Options) (*tls.Config, error) {
	certificate, err := tls.LoadX509KeyPair(options.ClientCertificatePath, options.ClientKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load Remote Write client certificate: %w", err)
	}

	rootCAs, err := loadRootCAs(cfg.Heartbeat.CAPath)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      rootCAs,
		//nolint:gosec // Configuration intentionally controls server certificate validation.
		InsecureSkipVerify: !cfg.Heartbeat.TLSVerify,
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

	if caPath == "" || caPath == "system" {
		return rootCAs, nil
	}

	caData, err := (fs.Filesystem{}).Read(caPath)
	if err != nil {
		return nil, fmt.Errorf("read Remote Write CA certificate %s: %w", caPath, err)
	}

	if !rootCAs.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("read Remote Write CA certificate %s: no certificates found", caPath)
	}

	return rootCAs, nil
}

// configureProxy applies the configured proxy URI and credentials to the transport.
func configureProxy(transport *http.Transport, proxy config.Proxy) error {
	if proxy.URI.String() == "" {
		if proxy.Username != "" || proxy.Password != "" {
			return errors.New("configure HTTP proxy: credentials require a proxy URI")
		}

		return nil
	}

	proxyURL := proxy.URI
	if proxyURL.Hostname() == "" || (proxyURL.Scheme != "http" && proxyURL.Scheme != "https") {
		return errors.New("configure HTTP proxy: invalid proxy URL")
	}

	if proxyURL.User != nil {
		return errors.New("configure HTTP proxy: credentials must not be embedded in the proxy URI")
	}

	if proxy.Username != "" || proxy.Password != "" {
		proxyURL.User = url.UserPassword(proxy.Username, proxy.Password)
	}

	transport.Proxy = http.ProxyURL(&proxyURL)

	return nil
}
