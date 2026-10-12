package config

import "testing"

func TestDefaultHasNoNils(t *testing.T) {
	t.Parallel()

	partial, err := newPartialFromDefaults()
	if err != nil {
		t.Fatal(err)
	}

	if partial.API.Heartbeat.URI == nil {
		t.Fatal("API.Heartbeat.URI must not be nil")
	}

	if partial.API.Heartbeat.CAPath == nil {
		t.Fatal("API.Heartbeat.CAPath must not be nil")
	}

	if partial.API.Heartbeat.TLSVerify == nil {
		t.Fatal("API.Heartbeat.TLSVerify must not be nil")
	}

	if partial.HTTP.Timeout.Connect == nil {
		t.Fatal("HTTP.Timeout.Connect must not be nil")
	}

	if partial.HTTP.Timeout.Request == nil {
		t.Fatal("HTTP.Timeout.Request must not be nil")
	}

	if partial.HTTP.Timeout.Idle == nil {
		t.Fatal("HTTP.Timeout.Idle must not be nil")
	}

	if partial.HTTP.Proxy.URI == nil {
		t.Fatal("HTTP.Proxy.URI must not be nil")
	}

	if partial.HTTP.Proxy.NoProxy == nil {
		t.Fatal("HTTP.Proxy.NoProxy must not be nil")
	}
}
