package config

import (
	"testing"

	"github.com/m-horky/rhc-heartbeat/internal/config/rhsm"
)

// TestTelemetryURIMapsRedHatDomains verifies DNS-label-aware endpoint derivation.
//
// Given stage, production, and unrelated Candlepin hosts, when telemetryURI derives an endpoint,
// then only matching domain-label suffixes map to Red Hat telemetry services.
func TestTelemetryURIMapsRedHatDomains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		host string
		want string
	}{
		{"api.rhsm.stage.redhat.com", "https://cert.console.stage.redhat.com/api/rhel-telemetry/v1/receive"},
		{"RHSM.REDHAT.COM.", "https://cert.console.redhat.com/api/rhel-telemetry/v1/receive"},
		{"notrhsm.redhat.com", "https://notrhsm.redhat.com:443/prometheus/api/v1/write"},
		{"satellite.example.com", "https://satellite.example.com:8443/prometheus/api/v1/write"},
	}
	for _, test := range tests {
		t.Run(test.host, func(t *testing.T) {
			server := rhsm.Server{Hostname: test.host, Port: 8443}
			if test.host == "notrhsm.redhat.com" {
				server.Port = 443
			}

			got, err := buildTelemetryURI(server)
			if err != nil {
				t.Fatalf("telemetryURI() error = %v", err)
			}

			if got.String() != test.want {
				t.Errorf("telemetryURI() = %q, want %q", got.String(), test.want)
			}
		})
	}
}

// TestTelemetryURIRejectsInteriorEmptyLabels verifies malformed domains do not match by suffix.
//
// Given a Candlepin hostname containing an empty DNS label, when telemetryURI derives its endpoint,
// then it returns a hostname validation error.
func TestTelemetryURIRejectsInteriorEmptyLabels(t *testing.T) {
	t.Parallel()

	if _, err := buildTelemetryURI(rhsm.Server{Hostname: "subscription..rhsm.redhat.com", Port: 443}); err == nil {
		t.Fatal("telemetryURI() error = nil, want invalid hostname error")
	}
}
