package remotewrite

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang/snappy"
	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"github.com/m-horky/rhc-heartbeat/pkg/version"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

// TestNewRejectsInvalidEndpoint verifies invalid Remote Write endpoint URIs are rejected before use.
//
// Given an empty or malformed endpoint URI, when constructing the client, then it returns a validation error.
func TestNewRejectsInvalidEndpoint(t *testing.T) {
	t.Parallel()

	for _, endpoint := range []string{"", "://broken", "ftp://prometheus.example.com/write"} {
		if _, err := New(config.Config{Heartbeat: config.Endpoint{URI: endpoint}}); err == nil {
			t.Errorf("New(%q) error = nil, want invalid endpoint error", endpoint)
		}
	}
}

// TestUploadSendsSnappyProtobufAndRequiredHeaders verifies Remote Write v1 request encoding and headers.
//
// Given a valid heartbeat and endpoint, when Upload sends the batch, then it sends block-compressed
// protobuf with the required protocol headers.
func TestUploadSendsSnappyProtobufAndRequiredHeaders(t *testing.T) {
	t.Parallel()

	requests := make(chan http.Header, 1)
	schema := newRemoteWriteMessages()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests <- request.Header.Clone()

		payload, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)

			return
		}

		decoded, err := snappy.Decode(nil, payload)
		if err != nil {
			t.Errorf("decode Snappy block: %v", err)

			return
		}

		message := dynamicpb.NewMessage(schema.writeRequest)
		if err := proto.Unmarshal(decoded, message); err != nil {
			t.Errorf("decode protobuf WriteRequest: %v", err)
		}

		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(config.Config{Heartbeat: config.Endpoint{URI: server.URL}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.CloseIdleConnections()

	hb := heartbeat.Heartbeat{HostID: "uuid", HostOrg: "org", BootID: "boot", Kind: heartbeat.KindOff}
	if err := client.Upload(context.Background(), []heartbeat.Heartbeat{hb}); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	headers := <-requests
	if got := headers.Get("Content-Encoding"); got != "snappy" {
		t.Errorf("Content-Encoding = %q, want snappy", got)
	}

	if got := headers.Get("Content-Type"); got != "application/x-protobuf" {
		t.Errorf("Content-Type = %q, want application/x-protobuf", got)
	}

	if got := headers.Get("X-Prometheus-Remote-Write-Version"); got != "0.1.0" {
		t.Errorf("Remote Write version = %q, want 0.1.0", got)
	}

	if got := headers.Get("User-Agent"); !strings.Contains(got, "rhc-heartbeat/"+version.Version) {
		t.Errorf("User-Agent = %q, want rhc-heartbeat version", got)
	}
}

// TestUploadReturnsErrorForNonSuccessStatus verifies failed receiver responses are reported.
//
// Given a receiver returning HTTP 503, when Upload sends a heartbeat, then it returns an error
// without interpreting the response body.
func TestUploadReturnsErrorForNonSuccessStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = writer.Write([]byte("receiver error"))
	}))
	defer server.Close()

	client, err := New(config.Config{Heartbeat: config.Endpoint{URI: server.URL}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.CloseIdleConnections()

	hb := heartbeat.Heartbeat{HostID: "uuid", HostOrg: "org", BootID: "boot", Kind: heartbeat.KindPing}

	err = client.Upload(context.Background(), []heartbeat.Heartbeat{hb})
	if err == nil || !strings.Contains(err.Error(), "503 Service Unavailable") {
		t.Errorf("Upload() error = %v, want HTTP status error", err)
	}
}
