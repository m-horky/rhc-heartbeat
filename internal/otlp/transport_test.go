package otlp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"go.opentelemetry.io/collector/pdata/plog"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip invokes the test function as an HTTP transport.
func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

// TestNewRejectsInvalidEndpoint verifies invalid OTLP endpoints are rejected before use.
//
// Given an empty or malformed endpoint URI
// When an OTLP client is constructed
// Then construction returns an error.
func TestNewRejectsInvalidEndpoint(t *testing.T) {
	t.Parallel()

	for _, endpoint := range []string{"", "ftp://collector.example/v1/logs", "https://user@example.test/v1/logs"} {
		t.Run(endpoint, func(t *testing.T) {
			t.Parallel()

			if _, err := New(config.Config{OTEL: config.Endpoint{URI: endpoint, TLSVerify: true}}); err == nil {
				t.Fatal("New() error = nil, want invalid endpoint error")
			}
		})
	}
}

// TestUploadPostsOTLPJSON verifies a heartbeat batch is sent as OTLP/JSON.
//
// Given a configured HTTP endpoint and heartbeat batch
// When the OTLP client uploads the batch
// Then it posts a valid OTLP/JSON payload and accepts an empty partial-success response.
func TestUploadPostsOTLPJSON(t *testing.T) {
	t.Parallel()

	var received bool

	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		received = true

		if request.Method != http.MethodPost {
			t.Errorf("request method = %q, want POST", request.Method)
		}

		if request.URL.Path != "/v1/logs" {
			t.Errorf("request path = %q, want /v1/logs", request.URL.Path)
		}

		if request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", request.Header.Get("Content-Type"))
		}

		body, err := io.ReadAll(request.Body)
		if err != nil {
			return nil, fmt.Errorf("read test request body: %w", err)
		}

		if _, err := (&plog.JSONUnmarshaler{}).UnmarshalLogs(body); err != nil {
			t.Errorf("decode OTLP/JSON request: %v", err)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"partialSuccess":{}}`)),
			Request:    request,
		}, nil
	})

	client := &Client{
		endpoint: "http://collector.example/v1/logs",
		http:     &http.Client{Transport: transport},
	}
	if err := client.Upload(context.Background(), []heartbeat.Heartbeat{{HostID: "host-123"}}); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	if !received {
		t.Fatal("transport was not called")
	}
}

// TestUploadReportsHTTPAndPartialSuccessFailures verifies uncertain and rejected deliveries return errors.
//
// Given an OTLP endpoint response with an HTTP failure or rejected records
// When the client uploads a heartbeat
// Then it returns an error so the processor can retain the heartbeat for retry.
func TestUploadReportsHTTPAndPartialSuccessFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "http failure", statusCode: http.StatusServiceUnavailable, body: "unavailable"},
		{
			name:       "partial success",
			statusCode: http.StatusOK,
			body:       `{"partialSuccess":{"rejectedLogRecords":"1","errorMessage":"rejected"}}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			client := &Client{
				endpoint: "http://collector.example/v1/logs",
				http: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: test.statusCode,
						Status:     fmt.Sprintf("%d %s", test.statusCode, http.StatusText(test.statusCode)),
						Header:     make(http.Header),
						Body:       io.NopCloser(strings.NewReader(test.body)),
						Request:    request,
					}, nil
				})},
			}

			if err := client.Upload(context.Background(), []heartbeat.Heartbeat{{HostID: "host-123"}}); err == nil {
				t.Fatal("Upload() error = nil, want delivery error")
			}
		})
	}
}

// TestUploadSkipsEmptyBatch verifies empty requests do not contact the endpoint.
//
// Given an empty heartbeat batch
// When the client uploads the batch
// Then it returns without making an HTTP request.
func TestUploadSkipsEmptyBatch(t *testing.T) {
	t.Parallel()

	client := &Client{
		endpoint: "http://collector.example/v1/logs",
		http: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("unexpected HTTP request")
		})},
	}

	if err := client.Upload(context.Background(), nil); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
}

// TestNewHTTPClientUsesConfiguredProxy verifies resolved proxy URI and credentials reach the transport.
//
// Given configuration with a proxy URI and credentials
// When the OTLP HTTP client is constructed
// Then its transport uses that proxy with the configured credentials.
func TestNewHTTPClientUsesConfiguredProxy(t *testing.T) {
	t.Parallel()

	endpoint := "http://collector.example/v1/logs"
	cfg := config.Config{
		OTEL: config.Endpoint{URI: endpoint, TLSVerify: true},
		HTTP: config.HTTPConfig{Proxy: config.Proxy{
			URI:      "https://proxy.example:8080",
			User:     "user",
			Password: "secret",
		}},
	}

	client, err := newHTTPClient(cfg, endpoint)
	if err != nil {
		t.Fatalf("newHTTPClient() error = %v", err)
	}

	transport := client.Transport.(*http.Transport)

	proxyURL, err := transport.Proxy(&http.Request{URL: mustParseURL(t, endpoint)})
	if err != nil {
		t.Fatalf("transport proxy lookup error = %v", err)
	}

	if proxyURL == nil || proxyURL.Scheme != "https" || proxyURL.Host != "proxy.example:8080" {
		t.Fatalf("proxy URL = %v, want configured HTTPS proxy", proxyURL)
	}

	if proxyURL.User == nil {
		t.Fatal("proxy URL has no credentials")
	}

	password, ok := proxyURL.User.Password()
	if proxyURL.User.Username() != "user" || !ok || password != "secret" {
		t.Fatalf("proxy credentials = %v, want user:secret", proxyURL.User)
	}
}

// TestConfigureProxyRejectsCredentialsOverHTTP verifies credentials cannot be sent through a cleartext proxy.
//
// Given an HTTP proxy with credentials configured in fields or embedded in its URI
// When the proxy is configured
// Then configuration fails rather than sending credentials without TLS.
func TestConfigureProxyRejectsCredentialsOverHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		proxy config.Proxy
	}{
		{
			name:  "separate credentials",
			proxy: config.Proxy{URI: "http://proxy.example:8080", User: "user", Password: "secret"},
		},
		{
			name:  "username without password",
			proxy: config.Proxy{URI: "http://proxy.example:8080", User: "user"},
		},
		{
			name:  "embedded credentials",
			proxy: config.Proxy{URI: strings.Join([]string{"http://user", "secret@proxy.example:8080"}, ":")},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := configureProxy(&http.Transport{}, test.proxy); err == nil {
				t.Fatal("configureProxy() error = nil, want cleartext credential error")
			}
		})
	}
}

// TestNewHTTPClientIgnoresEnvironmentProxy verifies environment proxies are not inherited.
//
// Given no proxy configured in the application settings
// When the OTLP HTTP client is constructed
// Then its transport has no proxy configured from the process environment.
func TestNewHTTPClientIgnoresEnvironmentProxy(t *testing.T) {
	t.Parallel()

	endpoint := "http://collector.example/v1/logs"

	client, err := newHTTPClient(config.Config{OTEL: config.Endpoint{URI: endpoint}}, endpoint)
	if err != nil {
		t.Fatalf("newHTTPClient() error = %v", err)
	}

	transport := client.Transport.(*http.Transport)
	if transport.Proxy != nil {
		t.Fatal("transport.Proxy is configured, want environment proxy disabled")
	}
}

// TestPreventCrossOriginRedirectRejectsCrossOrigin verifies redirects cannot change the request origin.
//
// Given a redirect that changes the scheme, host, or port
// When the redirect policy is evaluated
// Then the redirect is rejected, while a same-origin redirect is allowed.
func TestPreventCrossOriginRedirectRejectsCrossOrigin(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name         string
		previousURLs []string
		destination  string
		wantError    bool
	}{
		{
			name:         "same origin with default port",
			previousURLs: []string{"https://collector.example/v1/logs"},
			destination:  "https://collector.example:443/other",
		},
		{
			name:         "different host",
			previousURLs: []string{"https://collector.example/v1/logs"},
			destination:  "https://other.example/v1/logs",
			wantError:    true,
		},
		{
			name:         "different port",
			previousURLs: []string{"https://collector.example/v1/logs"},
			destination:  "https://collector.example:8443/v1/logs",
			wantError:    true,
		},
		{
			name:         "HTTPS to HTTP",
			previousURLs: []string{"https://collector.example/v1/logs"},
			destination:  "http://collector.example/v1/logs",
			wantError:    true,
		},
		{
			name:         "HTTP to HTTPS to HTTP",
			previousURLs: []string{"http://collector.example/v1/logs", "https://collector.example/v1/logs"},
			destination:  "http://collector.example/final",
			wantError:    true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			previousRequests := make([]*http.Request, 0, len(test.previousURLs))
			for _, previousURL := range test.previousURLs {
				previousRequests = append(previousRequests, &http.Request{URL: mustParseURL(t, previousURL)})
			}

			next := &http.Request{URL: mustParseURL(t, test.destination)}

			err := preventCrossOriginRedirect(next, previousRequests)
			if (err != nil) != test.wantError {
				t.Errorf("preventCrossOriginRedirect() error = %v, wantError %t", err, test.wantError)
			}
		})
	}
}

// mustParseURL parses a URL for test request construction and fails the test on invalid input.
func mustParseURL(t *testing.T, value string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", value, err)
	}

	return parsed
}

// TestUploadUsesObservedTimeForEveryRecord verifies batch records share the upload observation time.
//
// Given a batch containing multiple heartbeats
// When logs are formatted at one observation time
// Then each OTLP log record contains that same observed timestamp.
func TestUploadUsesObservedTimeForEveryRecord(t *testing.T) {
	t.Parallel()

	observedAt := time.Unix(1_700_000_001, 0)

	logs := buildLogs([]heartbeat.Heartbeat{{}, {}}, observedAt)
	for i := 0; i < logs.ResourceLogs().Len(); i++ {
		record := logs.ResourceLogs().At(i).ScopeLogs().At(0).LogRecords().At(0)
		if !record.ObservedTimestamp().AsTime().Equal(observedAt) {
			t.Errorf("record %d observed time = %s, want %s", i, record.ObservedTimestamp().AsTime(), observedAt)
		}
	}
}
