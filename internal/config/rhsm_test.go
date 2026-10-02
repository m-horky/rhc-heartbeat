package config

import (
	"strings"
	"testing"
)

// TestMapRHSMBuildsOTELURIFromCandlepin verifies RHSM fields are translated to endpoint and proxy settings.
//
// Given server and proxy settings, when mapping runs, then it builds origin-based URIs and carries TLS data.
func TestMapRHSMBuildsOTELURIFromCandlepin(t *testing.T) {
	t.Parallel()

	got, err := mapRHSM(rhsmConfiguration{
		Server: rhsmServer{Hostname: "satellite.example.com", Port: "8443", Insecure: "yes"},
		RHSM:   rhsmRHSM{RepoCACert: " /etc/pki/satellite-ca.pem "},
		Proxy:  rhsmProxy{Hostname: "proxy.example.com", Port: "3128", User: " site-user ", Password: "secret"},
	})
	if err != nil {
		t.Fatalf("mapRHSM() error = %v", err)
	}

	if got.CandlepinURI != "https://satellite.example.com:8443" {
		t.Errorf("CandlepinURI = %q", got.CandlepinURI)
	}

	if got.OTELURI != "https://satellite.example.com:8443/otel/v1/logs" {
		t.Errorf("OTELURI = %q", got.OTELURI)
	}

	if !got.Insecure || !got.InsecurePresent {
		t.Errorf("Insecure = (%v, %v), want (true, true)", got.Insecure, got.InsecurePresent)
	}

	if got.CAPath != "/etc/pki/satellite-ca.pem" {
		t.Errorf("CAPath = %q", got.CAPath)
	}

	proxy := got.Proxy
	if proxy.URI != "https://proxy.example.com:3128" ||
		proxy.User != "site-user" || proxy.Password != "secret" {
		t.Errorf("Proxy = %+v", proxy)
	}
}

// TestMapRHSMSupportsIPv6AndNoPrefix verifies Candlepin and OTEL URI construction for IPv6 hosts.
//
// Given an IPv6 hostname without a prefix, when mapping runs, then it brackets the host and appends the logs path.
func TestMapRHSMSupportsIPv6AndNoPrefix(t *testing.T) {
	t.Parallel()

	got, err := mapRHSM(rhsmConfiguration{Server: rhsmServer{Hostname: "2001:db8::1", Port: "443"}})
	if err != nil {
		t.Fatalf("mapRHSM() error = %v", err)
	}

	if got.CandlepinURI != "https://[2001:db8::1]:443" || got.OTELURI != "https://[2001:db8::1]:443/otel/v1/logs" {
		t.Errorf("URIs = (%q, %q)", got.CandlepinURI, got.OTELURI)
	}
}

// TestMapRHSMRejectsInvalidServerAndProxySettings verifies malformed RHSM values are attributed to their keys.
//
// Given incomplete or invalid server and proxy fields, when mapping runs, then it reports the relevant key.
func TestMapRHSMRejectsInvalidServerAndProxySettings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		conf rhsmConfiguration
		key  string
	}{
		{
			name: "missing hostname",
			conf: rhsmConfiguration{Server: rhsmServer{Port: "443"}},
			key:  "server.hostname",
		},
		{
			name: "missing port",
			conf: rhsmConfiguration{Server: rhsmServer{Hostname: "example.com"}},
			key:  "server.port",
		},
		{
			name: "invalid hostname",
			conf: rhsmConfiguration{Server: rhsmServer{Hostname: "bad host", Port: "443"}},
			key:  "server.hostname",
		},
		{
			name: "invalid insecure",
			conf: rhsmConfiguration{Server: rhsmServer{Insecure: "sometimes"}},
			key:  "server.insecure",
		},
		{
			name: "missing proxy hostname",
			conf: rhsmConfiguration{Proxy: rhsmProxy{Port: "3128"}},
			key:  "proxy.proxy_hostname",
		},
		{
			name: "missing proxy port",
			conf: rhsmConfiguration{Proxy: rhsmProxy{Hostname: "proxy.example.com"}},
			key:  "proxy.proxy_port",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := mapRHSM(tt.conf)
			if err == nil || !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("mapRHSM() error = %v, want key %q", err, tt.key)
			}
		})
	}
}
