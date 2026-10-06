package config

import (
	"os"
	"path/filepath"
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

	rhsmConfig := "[server]\n" + "hostname = satellite.example.com\n" + "port = 8443\n"
	if err := os.WriteFile(rhsmPath, []byte(rhsmConfig), 0o600); err != nil {
		t.Fatalf("write RHSM configuration: %v", err)
	}

	t.Setenv(constants.ConfigPathEnv, configPath)
	t.Setenv(constants.RHSMPathEnv, rhsmPath)

	got, err := Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	want := "https://satellite.example.com:8443/api/v1/write"
	if got.Heartbeat.URI != want {
		t.Errorf("Get().Heartbeat.URI = %q, want %q", got.Heartbeat.URI, want)
	}
}
