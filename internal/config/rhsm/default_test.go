package rhsm

import "testing"

func TestDefaultHasNoNils(t *testing.T) {
	t.Parallel()

	partial, err := newPartialFromDefaults()
	if err != nil {
		t.Fatal(err)
	}

	if partial.Server.Hostname == nil {
		t.Fatal("partial.Server.Hostname is nil")
	}

	if partial.Server.Prefix == nil {
		t.Fatal("partial.Server.Prefix is nil")
	}

	if partial.Server.Port == nil {
		t.Fatal("partial.Server.Port is nil")
	}

	if partial.Server.Insecure == nil {
		t.Fatal("partial.Server.Insecure is nil")
	}

	if partial.Server.ProxyHostname == nil {
		t.Fatal("partial.Server.ProxyHostname is nil")
	}

	if partial.Server.ProxyScheme == nil {
		t.Fatal("partial.Server.ProxyScheme is nil")
	}

	if partial.Server.ProxyPort == nil {
		t.Fatal("partial.Server.ProxyPort is nil")
	}

	if partial.Server.ProxyUser == nil {
		t.Fatal("partial.Server.ProxyUser is nil")
	}

	if partial.Server.ProxyPassword == nil {
		t.Fatal("partial.Server.ProxyPassword is nil")
	}

	if partial.Server.NoProxy == nil {
		t.Fatal("partial.Server.NoProxy is nil")
	}

	if partial.RHSM.CACertDir == nil {
		t.Fatal("partial.RHSM.CACertDir is nil")
	}

	if partial.RHSM.RepoCACert == nil {
		t.Fatal("partial.RHSM.RepoCACert is nil")
	}
}
