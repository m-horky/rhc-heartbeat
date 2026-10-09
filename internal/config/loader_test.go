package config

import (
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/m-horky/rhc-heartbeat/internal/config/rhsm"
	internalfs "github.com/m-horky/rhc-heartbeat/internal/fs"
)

// TestLoadMergesNativeFilesAndDropIns verifies documented configuration precedence.
//
// Given defaults, a main file, and lexicographically ordered drop-ins, when Load resolves the configuration,
// then later layers win and valid timeout and bypass values are retained.
func TestLoadMergesNativeFilesAndDropIns(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mainPath := filepath.Join(root, "main.toml")

	dropInDir := filepath.Join(root, "drop-ins")
	if err := os.Mkdir(dropInDir, 0o700); err != nil {
		t.Fatal(err)
	}

	writeNativeTestFile(t, mainPath, `[api.heartbeat]
uri = "https://main.example.com/write"
[http.timeout]
connect = 0
[http.proxy]
no-proxy = ["*.example.com", "10.0.0.0/8", "bad**"]
`)
	writeNativeTestFile(t, filepath.Join(dropInDir, "10-first.conf"), `[api.heartbeat]
uri = "https://first.example.com/write"
`)
	writeNativeTestFile(t, filepath.Join(dropInDir, "20-last.CONF"), `[api.heartbeat]
uri = "https://last.example.com/write"
`)

	if err := os.Mkdir(filepath.Join(dropInDir, "ignored.conf"), 0o700); err != nil {
		t.Fatal(err)
	}

	fallback := testRHSMFallback()

	got, err := Load(internalfs.Filesystem{}, mainPath, dropInDir, &fallback)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.API.Heartbeat.URI.String() != "https://last.example.com/write" {
		t.Errorf("endpoint = %q, want final drop-in endpoint", got.API.Heartbeat.URI.String())
	}

	if got.HTTP.Timeout.Connect != 0 {
		t.Errorf("connect timeout = %s, want zero", got.HTTP.Timeout.Connect)
	}

	if want := []string{"*.example.com", "10.0.0.0/8"}; !reflect.DeepEqual(got.HTTP.Proxy.NoProxy, want) {
		t.Errorf("no-proxy = %#v, want %#v", got.HTTP.Proxy.NoProxy, want)
	}

	if got.API.Heartbeat.URI.Scheme != "https" {
		t.Errorf("endpoint scheme = %q, want https", got.API.Heartbeat.URI.Scheme)
	}

	if got.API.Heartbeat.TLSVerify || got.API.Heartbeat.CAPath != "/legacy/ca.pem" {
		t.Errorf("RHSM endpoint fallback = %+v, want TLS and CA fallback values", got.API.Heartbeat)
	}

	if got.HTTP.Proxy.URI.String() != "http://proxy.example.com:3128" ||
		got.HTTP.Proxy.Username != "legacy-user" || got.HTTP.Proxy.Password != "legacy-password" {
		t.Error("RHSM proxy fallback did not preserve the expected URI and credentials")
	}
}

// TestLoadRejectsExplicitEmptyEndpoint verifies empty supplied URIs are not silently suppressed.
//
// Given a valid default endpoint and an explicitly empty endpoint in the main file, when Load resolves,
// then it reports an invalid configuration instead of restoring the default.
func TestLoadRejectsExplicitEmptyEndpoint(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mainPath := filepath.Join(root, "main.toml")
	writeNativeTestFile(t, mainPath, "[api.heartbeat]\nuri = \"\"\n")

	_, err := Load(internalfs.Filesystem{}, mainPath, filepath.Join(root, "missing"), nil)
	if err == nil {
		t.Fatal("Load() error = nil, want explicit empty endpoint failure")
	}
}

// TestLoadAllowsMissingSources verifies absent native files and drop-in directories use defaults.
//
// Given missing optional native sources, when Load runs, then complete embedded defaults are returned.
func TestLoadAllowsMissingSources(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	got, err := Load(internalfs.Filesystem{}, filepath.Join(root, "missing.toml"), filepath.Join(root, "missing.d"), nil)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want, err := url.Parse("https://cert.console.redhat.com/api/rhel-telemetry/v1/receive")
	if err != nil {
		t.Fatal(err)
	}

	if got.API.Heartbeat.URI.String() != want.String() {
		t.Errorf("default endpoint = %q, want %q", got.API.Heartbeat.URI.String(), want.String())
	}

	if got.HTTP.Proxy.NoProxy == nil {
		t.Error("default no-proxy list is nil, want initialized empty list")
	}
}

// testRHSMFallback returns deterministic legacy values for native-merge tests.
func testRHSMFallback() rhsm.Config {
	return rhsm.Config{
		Server: rhsm.Server{
			Hostname:      "satellite.example.com",
			Port:          443,
			TLSVerify:     false,
			ProxyHostname: "proxy.example.com",
			ProxyScheme:   "http",
			ProxyPort:     3128,
			ProxyUser:     "legacy-user",
			ProxyPassword: "legacy-password",
			NoProxy:       []string{"legacy.example.com"},
		},
		RHSM: rhsm.RHSM{RepoCACert: "/legacy/ca.pem"},
	}
}

// writeNativeTestFile writes one native configuration fixture.
func writeNativeTestFile(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write native test file: %v", err)
	}
}
