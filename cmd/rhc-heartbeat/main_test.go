package main

import (
	"bytes"
	"log/slog"
	"net/url"
	"strings"
	"testing"

	"github.com/m-horky/rhc-heartbeat/pkg/config"
)

// parseTestURL parses an endpoint used by a runtime warning test.
func parseTestURL(t *testing.T, value string) url.URL {
	t.Helper()

	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatalf("parse test URL: %v", err)
	}

	return *parsed
}

// TestWarnInsecureRemoteWriteTransport verifies warnings for unencrypted or unverified endpoints.
//
// Given endpoint transport settings, when the runtime warning check runs, then it warns only for
// enabled insecure transport modes.
func TestWarnInsecureRemoteWriteTransport(t *testing.T) {
	var output bytes.Buffer

	previousLogger := slog.Default()

	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })

	tests := []struct {
		name        string
		endpoint    string
		tlsVerify   bool
		wantWarning string
		absent      string
	}{
		{
			name:        "plain HTTP",
			endpoint:    "http://receiver.example.com/write",
			tlsVerify:   true,
			wantWarning: "plain HTTP; heartbeat data is not encrypted",
		},
		{
			name:        "HTTPS without certificate verification",
			endpoint:    "https://receiver.example.com/write",
			wantWarning: "TLS certificate verification is disabled",
		},
		{
			name:        "HTTP TLS setting is irrelevant",
			endpoint:    "http://receiver.example.com/write",
			wantWarning: "plain HTTP; heartbeat data is not encrypted",
			absent:      "TLS certificate verification is disabled",
		},
		{
			name:      "verified HTTPS",
			endpoint:  "https://receiver.example.com/write",
			tlsVerify: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output.Reset()
			logInsecureTransport(config.Config{
				Heartbeat: config.Endpoint{URI: parseTestURL(t, test.endpoint), TLSVerify: test.tlsVerify},
			})

			got := output.String()
			if test.wantWarning != "" && !strings.Contains(got, test.wantWarning) {
				t.Errorf("warning output = %q, want it to contain %q", got, test.wantWarning)
			}

			if test.absent != "" && strings.Contains(got, test.absent) {
				t.Errorf("warning output = %q, want it not to contain %q", got, test.absent)
			}

			if test.wantWarning == "" && got != "" {
				t.Errorf("warning output = %q, want no warning", got)
			}
		})
	}
}
