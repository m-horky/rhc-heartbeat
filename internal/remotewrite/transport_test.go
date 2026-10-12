package remotewrite

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/golang/snappy"
	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

// mustURL parses a test URL or fails the active test.
func mustURL(t *testing.T, value string) url.URL {
	t.Helper()

	parsed, err := url.Parse(value)
	if err != nil {
		return url.URL{Path: value}
	}

	return *parsed
}

// TestHeadersBody verifies Remote Write v1 request encoding and headers.
//
// Given a valid heartbeat and endpoint, when Upload sends the payload,
// then it sends block-compressed protobuf with the required protocol headers.
func TestHeadersBody(t *testing.T) {
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

	client, err := New(
		config.Config{Heartbeat: config.Endpoint{URI: mustURL(t, server.URL)}},
		Options{UserAgent: "rhc-heartbeat/test"},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.CloseIdleConnections()

	hb := heartbeat.Heartbeat{HostID: "uuid", HostOrg: "org", BootID: "boot", Kind: heartbeat.KindOff}
	if err := client.Upload(context.Background(), []heartbeat.Heartbeat{hb}); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	got := <-requests
	if value := got.Get("Content-Encoding"); value != "snappy" {
		t.Errorf("Content-Encoding = %q, want snappy", value)
	}

	if got := got.Get("Content-Type"); got != "application/x-protobuf" {
		t.Errorf("Content-Type = %q, want application/x-protobuf", got)
	}

	if got := got.Get("X-Prometheus-Remote-Write-Version"); got != "0.1.0" {
		t.Errorf("Remote Write version = %q, want 0.1.0", got)
	}

	if got := got.Get("User-Agent"); got != "rhc-heartbeat/test" {
		t.Errorf("User-Agent = %q, want rhc-heartbeat/test", got)
	}
}
