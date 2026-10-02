package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	heartbeatconfig "github.com/m-horky/rhc-heartbeat/pkg/config"
)

// TestConfigurationOutputRedactsProxyCredentials verifies serialized configuration does not expose proxy secrets.
//
// Given a resolved configuration with proxy credentials, when redacting and encoding it,
// then the output contains redaction markers instead of the original credentials.
func TestConfigurationOutputRedactsProxyCredentials(t *testing.T) {
	t.Parallel()

	cfg := heartbeatconfig.Config{
		HTTP: heartbeatconfig.HTTPConfig{
			Proxy: heartbeatconfig.Proxy{
				URI:      "https://proxy.example:8443",
				User:     "private-user",
				Password: "private-password",
			},
		},
	}

	var output bytes.Buffer
	if err := toml.NewEncoder(&output).Encode(redactProxyCredentials(cfg)); err != nil {
		t.Fatalf("encode redacted configuration: %v", err)
	}

	encoded := output.String()
	for _, secret := range []string{"private-user", "private-password"} {
		if strings.Contains(encoded, secret) {
			t.Errorf("configuration output contains secret %q", secret)
		}
	}

	if !strings.Contains(encoded, "[REDACTED]") {
		t.Errorf("configuration output = %q, want redaction markers", encoded)
	}

	if cfg.HTTP.Proxy.User != "private-user" || cfg.HTTP.Proxy.Password != "private-password" {
		t.Error("redacting output modified the resolved configuration")
	}
}
