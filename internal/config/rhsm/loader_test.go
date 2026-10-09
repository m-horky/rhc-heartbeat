package rhsm

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	internalfs "github.com/m-horky/rhc-heartbeat/internal/fs"
)

// TestLoadMergesRHSMOverridesAndExpandsCA verifies RHSM merging and deferred interpolation.
//
// Given embedded RHSM defaults and a file overriding insecure, proxy, and CA directory values,
// when Load resolves the file, then normalized TLS, proxy, no-proxy, and expanded CA values are returned.
func TestLoadMergesRHSMOverridesAndExpandsCA(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "rhsm.conf")

	contents := `[server]
hostname = telemetry.rhsm.stage.redhat.com
insecure = TRUE
proxy_hostname = proxy.example.com
proxy_scheme =
proxy_port =
no_proxy = example.com, 10.0.0.0/8,,
[rhsm]
ca_cert_dir = /custom/ca/
unknown = ignored
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(internalfs.Filesystem{}, path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Server.TLSVerify {
		t.Error("TLSVerify = true, want false for insecure=true")
	}

	if got.Server.ProxyScheme != "http" || got.Server.ProxyPort != 3128 {
		t.Errorf("proxy defaults = (%q, %d), want (http, 3128)", got.Server.ProxyScheme, got.Server.ProxyPort)
	}

	if want := []string{"example.com", "10.0.0.0/8"}; !reflect.DeepEqual(got.Server.NoProxy, want) {
		t.Errorf("no-proxy = %#v, want %#v", got.Server.NoProxy, want)
	}

	if got.RHSM.RepoCACert != "/custom/ca/redhat-uep.pem" {
		t.Errorf("repo CA = %q, want inherited path expanded against overridden directory", got.RHSM.RepoCACert)
	}
}

// TestLoadUsesDefaultsWhenRHSMFileIsMissing verifies a deleted RHSM file has usable defaults.
//
// Given no RHSM file, when Load resolves the embedded values, then the production subscription defaults are returned.
func TestLoadUsesDefaultsWhenRHSMFileIsMissing(t *testing.T) {
	t.Parallel()

	got, err := Load(internalfs.Filesystem{}, filepath.Join(t.TempDir(), "missing.conf"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.Server.Hostname != "subscription.rhsm.redhat.com" || got.Server.Port != 443 || !got.Server.TLSVerify {
		t.Errorf("default RHSM server = %+v, want secure subscription defaults", got.Server)
	}
}
