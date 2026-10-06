package upload

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"github.com/m-horky/rhc-heartbeat/pkg/version"
)

// TestNewUsesApplicationUserAgent verifies the public upload client supplies the application user agent.
//
// Given a valid HTTP Remote Write endpoint, when the public client uploads a heartbeat,
// then the request carries the application name and build version as its user agent.
func TestNewUsesApplicationUserAgent(t *testing.T) {
	t.Parallel()

	userAgent := make(chan string, 1)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		userAgent <- request.UserAgent()

		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(config.Config{Heartbeat: config.Endpoint{URI: server.URL}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.CloseIdleConnections()

	hb := heartbeat.Heartbeat{
		HostID:  "host",
		HostOrg: "org",
		BootID:  "boot",
		Kind:    heartbeat.KindPing,
	}
	if err := client.Upload(context.Background(), []heartbeat.Heartbeat{hb}); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	if got, want := <-userAgent, "rhc-heartbeat/"+version.Version; got != want {
		t.Errorf("User-Agent = %q, want %q", got, want)
	}
}
