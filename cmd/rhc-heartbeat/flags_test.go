package main

import (
	"testing"

	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

// TestParseKind verifies the command accepts only supported heartbeat kinds and defaults to ping.
//
// Given command-line arguments with a supported kind or malformed input, when the kind is parsed
// then a supported enum value is returned and unsupported values are rejected.
func TestParseKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		want      heartbeat.Kind
		wantError bool
	}{
		{name: "default", want: heartbeat.KindPing},
		{name: "on", args: []string{"--kind=on"}, want: heartbeat.KindOn},
		{name: "off", args: []string{"--kind", "off"}, want: heartbeat.KindOff},
		{name: "ping", args: []string{"--kind=ping"}, want: heartbeat.KindPing},
		{name: "arbitrary kind", args: []string{"--kind=systemd:custom event / 123"}, wantError: true},
		{name: "empty kind", args: []string{"--kind="}, wantError: true},
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
