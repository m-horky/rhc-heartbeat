package main

import (
	"errors"
	"flag"
	"path/filepath"
	"testing"

	"github.com/m-horky/rhc-heartbeat/pkg/cache"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

// TestParseKind verifies the command accepts arbitrary heartbeat kind strings with ping as the default.
//
// Given command-line arguments with the default or a custom kind
// When the kind is parsed
// Then the supplied value is returned unchanged, while malformed arguments are rejected.
func TestParseKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		want      heartbeat.Kind
		wantError bool
	}{
		{name: "default", want: heartbeat.KindPing},
		{name: "ping", args: []string{"--kind=ping"}, want: heartbeat.KindPing},
		{name: "off", args: []string{"--kind", "off"}, want: heartbeat.KindOff},
		{name: "arbitrary kind", args: []string{"--kind=systemd:custom event / 123"}, want: "systemd:custom event / 123"},
		{name: "unexpected positional argument", args: []string{"unexpected"}, wantError: true},
		{name: "unknown option", args: []string{"--unknown"}, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseKind(test.args)
			if test.wantError {
				if err == nil {
					t.Fatal("parseKind() error = nil, want error")
				}

				return
			}

			if err != nil {
				t.Fatalf("parseKind() error = %v", err)
			}

			if got != test.want {
				t.Fatalf("parseKind() = %q, want %q", got, test.want)
			}
		})
	}
}

// TestParseKindHelp verifies the help option is surfaced for successful usage output.
//
// Given the help flag
// When command arguments are parsed
// Then the parser reports the standard help sentinel without treating it as invalid input.
func TestParseKindHelp(t *testing.T) {
	t.Parallel()

	if _, err := parseKind([]string{"--help"}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("parseKind() error = %v, want flag.ErrHelp", err)
	}
}

// TestCacheAfterSetupFailureRetainsHeartbeat verifies setup failures fall back to persistent caching.
//
// Given a collected heartbeat and a cache with an available parent directory
// When processor setup fails
// Then the heartbeat is persisted and no further error is returned.
func TestCacheAfterSetupFailureRetainsHeartbeat(t *testing.T) {
	t.Parallel()

	pending := cache.New(filepath.Join(t.TempDir(), "heartbeat.jsonl"))
	want := heartbeat.Heartbeat{HostID: "host-123", Kind: heartbeat.KindPing}

	if err := cacheAfterSetupFailure(pending, want, errors.New("setup failed")); err != nil {
		t.Fatalf("cacheAfterSetupFailure() error = %v, want nil after successful cache", err)
	}

	got, err := pending.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(got) != 1 || got[0] != want {
		t.Fatalf("ReadAll() = %#v, want one cached heartbeat %#v", got, want)
	}
}

// TestCacheAfterSetupFailureReportsCacheError verifies setup and persistence failures are both returned.
//
// Given a collected heartbeat and a cache whose parent directory is missing
// When processor setup fails and the heartbeat cannot be cached
// Then the returned error includes both the setup and cache failures.
func TestCacheAfterSetupFailureReportsCacheError(t *testing.T) {
	t.Parallel()

	pending := cache.New(filepath.Join(t.TempDir(), "missing", "heartbeat.jsonl"))
	setupErr := errors.New("setup failed")

	err := cacheAfterSetupFailure(pending, heartbeat.Heartbeat{}, setupErr)
	if !errors.Is(err, setupErr) {
		t.Fatalf("cacheAfterSetupFailure() error = %v, want setup error", err)
	}
}
