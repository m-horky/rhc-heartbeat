package command

import (
	"context"
	"testing"
)

// TestChronycTrackingCommandUsesAbsolutePath verifies chronyc execution cannot be redirected through PATH.
//
// Given a PATH that could contain an attacker-controlled chronyc, when constructing the approved invocation,
// then the command uses the fixed absolute executable path and fixed arguments.
func TestChronycTrackingCommandUsesAbsolutePath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	cmd := chronycTrackingCommand(context.Background())
	if cmd.Path != chronycExecutable {
		t.Fatalf("command path = %q, want %q", cmd.Path, chronycExecutable)
	}

	if len(cmd.Args) != 3 || cmd.Args[1] != "-c" || cmd.Args[2] != "tracking" {
		t.Errorf("command args = %q, want fixed tracking invocation", cmd.Args)
	}
}

// TestOSRunnerRejectsUnapprovedInvocations verifies unknown commands cannot reach the process executor.
//
// Given an executable or argument list that is not approved, when running it, then the request is rejected.
func TestOSRunnerRejectsUnapprovedInvocations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "subscription-manager", args: []string{"facts"}},
		{name: "chronyc", args: []string{"tracking"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := (OSRunner{}).Run(context.Background(), test.name, test.args...)
			if err == nil {
				t.Fatal("Run() error = nil, want unapproved invocation error")
			}
		})
	}
}
