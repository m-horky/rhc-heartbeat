package config

import (
	"strings"
	"testing"

	"github.com/m-horky/rhc-heartbeat/internal/config/rhsm"
)

// TestBuildTelemetryURI verifies Observatorium URL is guessed from Candlepin hostname.
//
// Given stage, production, and Satellite Candlepin hosts, when we derive an endpoint,
// then only matching domain-label suffixes map to Red Hat telemetry services.
func TestBuildTelemetryURI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		host string
		port uint16
		want string
	}{
		{"foo.rhsm.stage.redhat.com", 443, "https://cert.console.stage.redhat.com/api/rhel-telemetry/v1/receive"},
		{"subscription-legacy.rhsm.redhat.com", 443, "https://cert.console.redhat.com/api/rhel-telemetry/v1/receive"},
		{"SUBSCRIPTION.RHSM.REDHAT.COM", 443, "https://cert.console.redhat.com/api/rhel-telemetry/v1/receive"},
		{"notrhsm.redhat.com", 443, "https://notrhsm.redhat.com/prometheus/api/v1/write"},
		{"satellite.example.com", 8443, "https://satellite.example.com:8443/prometheus/api/v1/write"},
		{"19.6.satellite.internal.redhat.com", 443, "https://19.6.satellite.internal.redhat.com/prometheus/api/v1/write"},
		{"203.0.113.3", 8443, "https://203.0.113.3:8443/prometheus/api/v1/write"},
		{"2001:db8::1", 8443, "https://[2001:db8::1]:8443/prometheus/api/v1/write"},
	}
	for _, test := range tests {
		t.Run(test.host, func(t *testing.T) {
			server := rhsm.Server{Hostname: test.host, Port: test.port}

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

// TestHasLabelSuffix verifies we correctly match hostname suffix.
//
// Given a FQDN segment, when we compare it to another segment,
// 'true' is returned when they share the suffix.
func TestHasLabelSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		prefix []string
		ok     bool
	}{
		{[]string{"a"}, false},
		{[]string{"b", "a"}, false},
		{[]string{"c", "b", "a"}, true},
		{[]string{"d", "c", "b", "a"}, true},
		{[]string{"c", "d", "b", "a"}, false},
		{[]string{"c", "b", "a", "x"}, false},
	}

	prefix := []string{"c", "b", "a"}

	for _, test := range tests {
		t.Run(strings.Join(test.prefix, " "), func(t *testing.T) {
			ok := hasLabelSuffix(test.prefix, prefix)

			if ok != test.ok {
				t.Errorf("hasLabelSuffix() = %v, want %v", ok, test.ok)
			}
		})
	}
}
