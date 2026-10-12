package remotewrite

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
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

// Options contain application-specific values used to configure the Remote Write client.
type Options struct {
	Filesystem         fs.FS
	CACertificatePath  string
	CertificatePath    string
	CertificateKeyPath string
	UserAgent          string
	Insecure           bool
}

// Client sends heartbeat batches to a Prometheus Remote Write v1 endpoint.
type Client struct {
	endpoint  string
	http      *http.Client
	userAgent string
}

// New constructs a Remote Write client using resolved heartbeat configuration and supplied options.
func New(c config.Config, options Options) (*Client, error) {
	httpClient, err := newHTTPClient(options, c.Heartbeat.TLSVerify, &c.HTTP.Proxy.URI)
	if err != nil {
		return nil, err
	}

	return &Client{endpoint: c.Heartbeat.URI.String(), http: httpClient, userAgent: options.UserAgent}, nil
}

// Upload encodes and sends the supplied heartbeats as a Snappy-compressed Remote Write request.
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

// newHTTPClient creates a configured HTTP client.
func newHTTPClient(options Options, tls bool, proxy *url.URL) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = (&net.Dialer{Timeout: connectTimeout}).DialContext
	transport.IdleConnTimeout = 30 * time.Second

	if tls {
		tlsConfig, err := newTLSConfig(options)
		if err != nil {
			return nil, err
		}

		transport.TLSClientConfig = tlsConfig
	} else {
		slog.Debug("using insecure HTTP transport")
	}

	if proxy != nil && proxy.String() != "" {
		transport.Proxy = http.ProxyURL(proxy)
		slog.Debug("using proxy", "proxy", proxy.Redacted())
	}

	return &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
	}, nil
}

// newTLSConfig creates the endpoint TLS configuration, including configured certificate verification.
func newTLSConfig(options Options) (*tls.Config, error) {
	certificate, err := loadX509KeyPair(options.Filesystem, options.CertificatePath, options.CertificateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load Remote Write client certificate: %w", err)
	}

	rootCAs, err := loadX509CA(options.Filesystem, options.CACertificatePath)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		Certificates:       []tls.Certificate{certificate},
		RootCAs:            rootCAs,
		InsecureSkipVerify: options.Insecure, //nolint:gosec // Configurable for development purposes.
	}, nil
}

// loadX509KeyPair reimplements tls.LoadX509KeyPair while using the provided filesystem.
func loadX509KeyPair(filesystem fs.FS, certPath, keyPath string) (tls.Certificate, error) {
	cert, err := filesystem.Read(certPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("read certificate: %w", err)
	}

	key, err := filesystem.Read(keyPath)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("read certificate key: %w", err)
	}

	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("parse certificate pair: %w", err)
	}

	return certificate, nil
}

// loadX509CA selects between the system CA store or the provided CA certificate path.
func loadX509CA(filesystem fs.FS, caPath string) (*x509.CertPool, error) {
	if caPath == "system" {
		pool, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("use system CA certificates: %w", err)
		}

		return pool, nil
	}

	pool := x509.NewCertPool()

	raw, err := filesystem.Read(caPath)
	if err != nil {
		return nil, fmt.Errorf("read CA certificate %s: %w", caPath, err)
	}

	if ok := pool.AppendCertsFromPEM(raw); !ok {
		return nil, fmt.Errorf("use CA certificate: %s", caPath)
	}

	return pool, nil
}
