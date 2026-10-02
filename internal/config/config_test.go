package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadFromPathsUsesRHSMDefaultsAndApplicationOverrides verifies that TOML values override RHSM fallbacks.
//
// Given legacy and application settings for the same fields, when loading runs, then TOML values take precedence.
func TestLoadFromPathsUsesRHSMDefaultsAndApplicationOverrides(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "rhc-heartbeat.conf")
	rhsmPath := filepath.Join(dir, "rhsm.conf")
	writeConfigTestFile(t, rhsmPath, `[server]
hostname = satellite.example.com
port = 8443
prefix = /rhsm
insecure = yes

[proxy]
proxy_hostname = proxy.example.com
proxy_port = 3128
proxy_user = rhsm-user
proxy_password = rhsm-password
`)
	writeConfigTestFile(t, configPath, `[otel]
uri = "https://telemetry.example.com/custom/v1/logs"
tls-verify = true

[http.proxy]
user = "site-user"
`)

	got, err := LoadFromPaths(configPath, rhsmPath)
	if err != nil {
		t.Fatalf("LoadFromPaths() error = %v", err)
	}

	if got.OTEL.URI != "https://telemetry.example.com/custom/v1/logs" || !got.OTEL.TLSVerify {
		t.Errorf("OTEL = %+v, want explicit configuration", got.OTEL)
	}

	proxy := got.HTTP.Proxy
	if proxy.URI != "https://proxy.example.com:3128" ||
		proxy.User != "site-user" || proxy.Password != "rhsm-password" {
		t.Errorf("HTTP.Proxy = %+v, want explicit username over RHSM proxy defaults", proxy)
	}
}

// TestLoadFromPathsIgnoresCommentedOverrides verifies only active TOML values override RHSM settings.
//
// Given TOML with active and commented settings, when loading runs, then only active values override RHSM.
func TestLoadFromPathsIgnoresCommentedOverrides(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "rhc-heartbeat.conf")
	rhsmPath := filepath.Join(dir, "rhsm.conf")
	writeConfigTestFile(t, rhsmPath, `[server]
hostname = satellite.example.com
port = 8443
prefix = /rhsm
insecure = yes

[rhsm]
repo_ca_cert = /etc/pki/rhsm-ca.pem

[proxy]
proxy_hostname = proxy.example.com
proxy_port = 3128
proxy_user = rhsm-user
proxy_password = rhsm-password
`)
	writeConfigTestFile(t, configPath, `# [otel]
# uri = "https://commented.example.com/v1/logs"
[otel]
uri = "https://configured.example.com/v1/logs"
# tls-verify = true
# ca-path = "/commented/ca.pem"

[http.proxy]
# uri = "https://commented-proxy.example.com:3128"
# user = "commented-user"
# password = "commented-password"
`)

	got, err := LoadFromPaths(configPath, rhsmPath)
	if err != nil {
		t.Fatalf("LoadFromPaths() error = %v", err)
	}

	if got.OTEL.URI != "https://configured.example.com/v1/logs" {
		t.Errorf("OTEL.URI = %q, want active TOML value", got.OTEL.URI)
	}

	if got.OTEL.TLSVerify || got.OTEL.CAPath != "/etc/pki/rhsm-ca.pem" {
		t.Errorf("OTEL = %+v, want non-overridden RHSM values", got.OTEL)
	}

	if got.HTTP.Proxy.URI != "https://proxy.example.com:3128" ||
		got.HTTP.Proxy.User != "rhsm-user" || got.HTTP.Proxy.Password != "rhsm-password" {
		t.Errorf("HTTP.Proxy = %+v, want non-overridden RHSM values", got.HTTP.Proxy)
	}

	missingConfig, err := LoadFromPaths(filepath.Join(dir, "missing.conf"), rhsmPath)
	if err != nil {
		t.Fatalf("LoadFromPaths() with missing TOML error = %v", err)
	}

	if missingConfig.OTEL.URI != "https://satellite.example.com:8443/otel/v1/logs" ||
		missingConfig.OTEL.TLSVerify || missingConfig.OTEL.CAPath != "/etc/pki/rhsm-ca.pem" {
		t.Errorf("configuration with missing TOML = %+v, want RHSM-derived values", missingConfig)
	}
}

// TestLoadFromPathsUsesRHSMDerivedOTELURI verifies OTEL endpoint construction from Candlepin settings.
//
// Given a Candlepin host and API prefix, when loading runs, then it ignores the prefix and appends the OTEL logs path.
func TestLoadFromPathsUsesRHSMDerivedOTELURI(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rhsmPath := filepath.Join(dir, "rhsm.conf")
	writeConfigTestFile(t, rhsmPath, `[server]
hostname = satellite.example.com
port = 443
prefix = /redhat_access
`)

	got, err := LoadFromPaths(filepath.Join(dir, "missing.conf"), rhsmPath)
	if err != nil {
		t.Fatalf("LoadFromPaths() error = %v", err)
	}

	if got.OTEL.URI != "https://satellite.example.com:443/otel/v1/logs" {
		t.Errorf("OTEL.URI = %q", got.OTEL.URI)
	}

	if !got.OTEL.TLSVerify {
		t.Error("OTEL.TLSVerify = false, want secure default")
	}
}

// TestLoadFromPathsPreservesExplicitEmptyEndpoint verifies explicit empty endpoints suppress legacy fallback.
//
// Given a derived RHSM endpoint and an explicitly empty TOML URI, when loading runs, then the URI remains empty.
func TestLoadFromPathsPreservesExplicitEmptyEndpoint(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.conf")
	rhsmPath := filepath.Join(dir, "rhsm.conf")

	writeConfigTestFile(t, configPath, `[otel]
uri = ""
`)
	writeConfigTestFile(t, rhsmPath, `[server]
hostname = satellite.example.com
port = 443
`)

	got, err := LoadFromPaths(configPath, rhsmPath)
	if err != nil {
		t.Fatalf("LoadFromPaths() error = %v", err)
	}

	if got.OTEL.URI != "" {
		t.Errorf("OTEL.URI = %q, want explicit empty override", got.OTEL.URI)
	}
}

// TestLoadFromPathsAllowsMissingFiles verifies missing configuration files are optional.
//
// Given absent TOML and RHSM files, when loading runs, then it returns secure empty defaults without error.
func TestLoadFromPathsAllowsMissingFiles(t *testing.T) {
	t.Parallel()

	got, err := LoadFromPaths(filepath.Join(t.TempDir(), "missing.conf"), filepath.Join(t.TempDir(), "missing-rhsm.conf"))
	if err != nil {
		t.Fatalf("LoadFromPaths() error = %v", err)
	}

	if got.OTEL.URI != "" || got.HTTP.Proxy.URI != "" {
		t.Errorf("configuration = %+v, want empty configuration", got)
	}
}

// TestLoadFromPathsRejectsInvalidFilesAndURIs verifies malformed configuration and invalid values fail loading.
//
// Given invalid TOML, endpoint, proxy, or RHSM port data, when loading runs, then it returns a useful error.
func TestLoadFromPathsRejectsInvalidFilesAndURIs(t *testing.T) {
	t.Parallel()

	credentialedProxy := strings.Join([]string{"https://user", "pass@proxy.example.com"}, ":")

	tests := []struct {
		name       string
		configData string
		rhsmData   string
		wantError  string
	}{
		{
			name:       "invalid TOML",
			configData: "[otel\nuri = 'broken'",
			wantError:  "decode configuration",
		},
		{
			name:       "invalid endpoint",
			configData: "[otel]\nuri = 'ftp://telemetry.example.com'",
			wantError:  "otel.uri",
		},
		{
			name:       "invalid proxy",
			configData: "[http.proxy]\nuri = '" + credentialedProxy + "'",
			wantError:  "http.proxy.uri",
		},
		{
			name:      "invalid legacy port",
			rhsmData:  "[server]\nhostname = satellite.example.com\nport = 99999",
			wantError: "server.port",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.conf")
			rhsmPath := filepath.Join(dir, "rhsm.conf")

			if tt.configData != "" {
				writeConfigTestFile(t, configPath, tt.configData)
			}

			if tt.rhsmData != "" {
				writeConfigTestFile(t, rhsmPath, tt.rhsmData)
			}

			_, err := LoadFromPaths(configPath, rhsmPath)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("LoadFromPaths() error = %v, want text %q", err, tt.wantError)
			}
		})
	}
}

// writeConfigTestFile writes a private test configuration file.
func writeConfigTestFile(t *testing.T, path, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}
