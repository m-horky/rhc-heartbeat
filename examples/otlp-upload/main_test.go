package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-horky/rhc-heartbeat/internal/otlp"
	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"go.opentelemetry.io/collector/pdata/plog"
)

// TestOTLPClientUploadsToConfiguredEndpoint verifies the example can export using resolved endpoint configuration.
//
// Given an OTLP test endpoint and one heartbeat
// When the example's OTLP client uploads the heartbeat
// Then the endpoint receives a valid OTLP/JSON request and returns success.
func TestOTLPClientUploadsToConfiguredEndpoint(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("request method = %q, want POST", request.Method)
		}

		if request.URL.Path != "/v1/logs" {
			t.Errorf("request path = %q, want /v1/logs", request.URL.Path)
		}

		payload, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}

		if _, err := (&plog.JSONUnmarshaler{}).UnmarshalLogs(payload); err != nil {
			t.Errorf("decode OTLP/JSON request: %v", err)
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"partialSuccess":{}}`))
	}))
	defer server.Close()

	client, err := otlp.New(config.Config{OTEL: config.Endpoint{URI: server.URL + "/v1/logs", TLSVerify: true}})
	if err != nil {
		t.Fatalf("otlp.New() error = %v", err)
	}
	defer client.CloseIdleConnections()

	if err := client.Upload(t.Context(), []heartbeat.Heartbeat{{HostID: "host-123"}}); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
}
