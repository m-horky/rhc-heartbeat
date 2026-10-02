package command

import (
	"context"
	"testing"
)

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
