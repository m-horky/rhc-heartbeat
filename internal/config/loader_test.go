package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/m-horky/rhc-heartbeat/internal/config/rhsm"
	internalfs "github.com/m-horky/rhc-heartbeat/internal/fs"
)

// TestLoadHasDefaults verifies Load returns at least the default configuration.
//
// Given all files and directories are absent, when Load resolves the configuration,
// then it returns the default configuration.
func TestLoadHasDefaults(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	partial, err := newPartialFromDefaults()
	if err != nil {
		t.Fatal(err)
	}

	want, err := partial.resolve()
	if err != nil {
		t.Fatal(err)
	}

	got, err := Load(
		internalfs.Filesystem{},
		filepath.Join(root, "main.toml"),
		filepath.Join(root, "drop-ins"),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("Load() mismatch (-want +got): %s", diff)
	}
}

// TestLoadReadsMainAndDropIns verifies documented configuration precedence.
//
// Given defaults, a main file, and the drop-ins, when Load resolves the configuration,
// then later layers win.
func TestLoadReadsMainAndDropIns(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mainPath := filepath.Join(root, "rhc-heartbeat.conf")

	dropInsDir := filepath.Join(root, "rhc-heartbeat.conf.d")
	if err := os.Mkdir(dropInsDir, 0o700); err != nil {
		t.Fatal(err)
	}

	write(t, mainPath, "[http.timeout]\nconnect=10\nrequest=10\nidle=10\n")
	write(t, filepath.Join(dropInsDir, "20.conf"), "[http.timeout]\nrequest=20\nidle=20\n")
	write(t, filepath.Join(dropInsDir, "30.conf"), "[http.timeout]\nidle=30\n")
	write(t, filepath.Join(dropInsDir, "40.conf"), "[http.timeout]\n")
	write(t, filepath.Join(dropInsDir, "50.conf"), "")

	fallback := getRHSMconf()

	got, err := Load(internalfs.Filesystem{}, mainPath, dropInsDir, &fallback)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.HTTP.Timeout.Connect != 10*time.Second {
		t.Errorf("connect timeout = %s, want 10", got.HTTP.Timeout.Connect)
	}

	if got.HTTP.Timeout.Request != 20*time.Second {
		t.Errorf("request timeout = %s, want 20", got.HTTP.Timeout.Request)
	}

	if got.HTTP.Timeout.Idle != 30*time.Second {
		t.Errorf("idle timeout = %s, want 30", got.HTTP.Timeout.Idle)
	}

	if got.HTTP.Proxy.URI.Host != "rhsm.example.org:3128" {
		t.Errorf("proxy hostname = %s, want %s", got.HTTP.Proxy.URI.Host, "rhsm.example.org:3128")
	}
}

// TestLoadRejectsExplicitEmptyEndpoint verifies empty supplied URIs are not silently suppressed.
//
// Given an explicitly empty endpoint in the main file, when Load resolves,
// then it reports an invalid configuration.
func TestLoadRejectsExplicitEmptyEndpoint(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	mainPath := filepath.Join(root, "main.toml")
	write(t, mainPath, "[api.heartbeat]\nuri = \"\"\n")

	_, err := Load(internalfs.Filesystem{}, mainPath, filepath.Join(root, "missing"), nil)
	if err == nil {
		t.Fatal("Load() error = nil")
	}
}

// getRHSMconf returns deterministic legacy values for native-merge tests.
func getRHSMconf() rhsm.Config {
	return rhsm.Config{
		Server: rhsm.Server{
			Hostname:      "rhsm.example.org",
			Port:          443,
			TLSVerify:     false,
			ProxyHostname: "rhsm.example.org",
			ProxyScheme:   "http",
			ProxyPort:     3128,
			ProxyUser:     "",
			ProxyPassword: "",
			NoProxy:       "",
		},
		RHSM: rhsm.RHSM{RepoCACert: "/legacy/ca.pem"},
	}
}

// write writes contents into a file on disk.
func write(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write to disk: %v", err)
	}
}
