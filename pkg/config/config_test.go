package config

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-horky/rhc-heartbeat/pkg/constants"
)

// TestGetUsesConfiguredRHSMPath verifies the RHSM path environment override.
//
// Given a temporary RHSM configuration path, when Get loads configuration, then it uses
// the RHSM-derived Remote Write endpoint.
func TestGetUsesConfiguredRHSMPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "heartbeat.conf")
	rhsmPath := filepath.Join(dir, "rhsm.conf")

	rhsmConfig := "[server]\nhostname = satellite.example.com\nport = 8443\n"
	if err := os.WriteFile(rhsmPath, []byte(rhsmConfig), 0o600); err != nil {
		t.Fatalf("write RHSM configuration: %v", err)
	}

	t.Setenv(string(constants.ConfigPath), configPath)
	t.Setenv(string(constants.RHSMConfigPath), rhsmPath)

	got, err := Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	want := "https://satellite.example.com:8443/prometheus/api/v1/write"
	if got.Heartbeat.URI.String() != want {
		t.Errorf("Get().Heartbeat.URI = %q, want %q", got.Heartbeat.URI.String(), want)
	}
}

// TestSafeURL verifies diagnostic URLs omit user information, queries, and fragments.
//
// Given a URL containing credentials and query secrets, when safeURL formats it,
// then the output retains the endpoint path without exposing secret values.
func TestSafeURL(t *testing.T) {
	value, err := url.Parse("https://user:password@example.com/write?token=secret#fragment-secret")
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}

	got := safeURL(*value)
	if strings.Contains(got, "user") || strings.Contains(got, "password") || strings.Contains(got, "secret") {
		t.Errorf("safeURL() = %q, must not expose URL secrets", got)
	}

	if want := "https://example.com/write"; got != want {
		t.Errorf("safeURL() = %q, want %q", got, want)
	}
}
