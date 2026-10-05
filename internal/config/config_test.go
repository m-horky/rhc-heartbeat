package config

import (
	iofs "io/fs"
	"strings"
	"testing"

	internalfs "github.com/m-horky/rhc-heartbeat/internal/fs"
)

// TestLoadFromPathsUsesRHSMDefaultsAndApplicationOverrides verifies that TOML values override RHSM fallbacks.
//
// Given legacy and application settings for the same fields, when loading runs, then TOML values take precedence.
func TestLoadFromPathsUsesRHSMDefaultsAndApplicationOverrides(t *testing.T) {
	t.Parallel()

	configPath := "/virtual/rhc-heartbeat.conf"
	rhsmPath := "/virtual/rhsm.conf"
	filesystem := configTestFS{files: make(map[string][]byte)}
	writeConfigTestFile(filesystem.files, rhsmPath, `[server]
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
	writeConfigTestFile(filesystem.files, configPath, `[api.heartbeat]
uri = "https://telemetry.example.com/api/v1/write"
tls-verify = true

[http.proxy]
user = "site-user"
`)

	got, err := LoadFromPaths(filesystem, configPath, rhsmPath)
	if err != nil {
		t.Fatalf("LoadFromPaths() error = %v", err)
	}

	if got.Heartbeat.URI != "https://telemetry.example.com/api/v1/write" || !got.Heartbeat.TLSVerify {
		t.Errorf("Heartbeat = %+v, want explicit configuration", got.Heartbeat)
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

	configPath := "/virtual/rhc-heartbeat.conf"
	rhsmPath := "/virtual/rhsm.conf"
	filesystem := configTestFS{files: make(map[string][]byte)}
	writeConfigTestFile(filesystem.files, rhsmPath, `[server]
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
	writeConfigTestFile(filesystem.files, configPath, `# [api.heartbeat]
# uri = "https://commented.example.com/v1/logs"
[api.heartbeat]
uri = "https://configured.example.com/api/v1/write"
# tls-verify = true
# ca-path = "/commented/ca.pem"

[http.proxy]
# uri = "https://commented-proxy.example.com:3128"
# user = "commented-user"
# password = "commented-password"
`)

	got, err := LoadFromPaths(filesystem, configPath, rhsmPath)
	if err != nil {
		t.Fatalf("LoadFromPaths() error = %v", err)
	}

	if got.Heartbeat.URI != "https://configured.example.com/api/v1/write" {
		t.Errorf("Heartbeat.URI = %q, want active TOML value", got.Heartbeat.URI)
	}

	if got.Heartbeat.TLSVerify || got.Heartbeat.CAPath != "/etc/pki/rhsm-ca.pem" {
		t.Errorf("Heartbeat = %+v, want non-overridden RHSM values", got.Heartbeat)
	}

	if got.HTTP.Proxy.URI != "https://proxy.example.com:3128" ||
		got.HTTP.Proxy.User != "rhsm-user" || got.HTTP.Proxy.Password != "rhsm-password" {
		t.Errorf("HTTP.Proxy = %+v, want non-overridden RHSM values", got.HTTP.Proxy)
	}

	missingConfig, err := LoadFromPaths(filesystem, "/virtual/missing.conf", rhsmPath)
	if err != nil {
		t.Fatalf("LoadFromPaths() with missing TOML error = %v", err)
	}

	if missingConfig.Heartbeat.URI != "https://satellite.example.com:8443/api/v1/write" ||
		missingConfig.Heartbeat.TLSVerify || missingConfig.Heartbeat.CAPath != "/etc/pki/rhsm-ca.pem" {
		t.Errorf("configuration with missing TOML = %+v, want RHSM-derived values", missingConfig)
	}
}

// TestLoadFromPathsUsesRHSMRemoteWriteURI verifies Remote Write endpoint construction from Candlepin settings.
//
// Given a Candlepin host and API prefix, when loading runs, then it ignores the prefix and appends the write path.
func TestLoadFromPathsUsesRHSMRemoteWriteURI(t *testing.T) {
	t.Parallel()

	rhsmPath := "/virtual/rhsm.conf"
	filesystem := configTestFS{files: make(map[string][]byte)}
	writeConfigTestFile(filesystem.files, rhsmPath, `[server]
hostname = satellite.example.com
port = 443
prefix = /redhat_access
`)

	got, err := LoadFromPaths(filesystem, "/virtual/missing.conf", rhsmPath)
	if err != nil {
		t.Fatalf("LoadFromPaths() error = %v", err)
	}

	if got.Heartbeat.URI != "https://satellite.example.com:443/api/v1/write" {
		t.Errorf("Heartbeat.URI = %q", got.Heartbeat.URI)
	}

	if !got.Heartbeat.TLSVerify {
		t.Error("Heartbeat.TLSVerify = false, want secure default")
	}
}

// TestLoadFromPathsPreservesExplicitEmptyEndpoint verifies explicit empty endpoints suppress legacy fallback.
//
// Given a derived RHSM endpoint and an explicitly empty TOML URI, when loading runs, then the URI remains empty.
func TestLoadFromPathsPreservesExplicitEmptyEndpoint(t *testing.T) {
	t.Parallel()

	configPath := "/virtual/config.conf"
	rhsmPath := "/virtual/rhsm.conf"
	filesystem := configTestFS{files: make(map[string][]byte)}

	writeConfigTestFile(filesystem.files, configPath, `[api.heartbeat]
uri = ""
`)
	writeConfigTestFile(filesystem.files, rhsmPath, `[server]
hostname = satellite.example.com
port = 443
`)

	got, err := LoadFromPaths(filesystem, configPath, rhsmPath)
	if err != nil {
		t.Fatalf("LoadFromPaths() error = %v", err)
	}

	if got.Heartbeat.URI != "" {
		t.Errorf("Heartbeat.URI = %q, want explicit empty override", got.Heartbeat.URI)
	}
}

// TestLoadFromPathsAllowsMissingFiles verifies missing configuration files are optional.
//
// Given absent TOML and RHSM files, when loading runs, then it returns secure empty defaults without error.
func TestLoadFromPathsAllowsMissingFiles(t *testing.T) {
	t.Parallel()

	filesystem := configTestFS{files: make(map[string][]byte)}

	got, err := LoadFromPaths(filesystem, "/virtual/missing.conf", "/virtual/missing-rhsm.conf")
	if err != nil {
		t.Fatalf("LoadFromPaths() error = %v", err)
	}

	if got.Heartbeat.URI != "" || got.HTTP.Proxy.URI != "" {
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
			configData: "[api.heartbeat\nuri = 'broken'",
			wantError:  "decode configuration",
		},
		{
			name:       "invalid endpoint",
			configData: "[api.heartbeat]\nuri = 'ftp://telemetry.example.com'",
			wantError:  "api.heartbeat.uri",
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

			configPath := "/virtual/config.conf"
			rhsmPath := "/virtual/rhsm.conf"
			filesystem := configTestFS{files: make(map[string][]byte)}

			if tt.configData != "" {
				writeConfigTestFile(filesystem.files, configPath, tt.configData)
			}

			if tt.rhsmData != "" {
				writeConfigTestFile(filesystem.files, rhsmPath, tt.rhsmData)
			}

			_, err := LoadFromPaths(filesystem, configPath, rhsmPath)
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("LoadFromPaths() error = %v, want text %q", err, tt.wantError)
			}
		})
	}
}

type configTestFS struct {
	internalfs.FS

	files map[string][]byte
}

var _ internalfs.FS = configTestFS{}

// Read returns configured test contents or a virtual not-exist error.
func (filesystem configTestFS) Read(path string) ([]byte, error) {
	data, ok := filesystem.files[path]
	if !ok {
		return nil, &iofs.PathError{Op: "read", Path: path, Err: iofs.ErrNotExist}
	}

	return append([]byte(nil), data...), nil
}

// writeConfigTestFile adds configuration contents to the in-memory test filesystem.
func writeConfigTestFile(files map[string][]byte, path, content string) {
	files[path] = []byte(content)
}
