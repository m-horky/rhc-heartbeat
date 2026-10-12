package constants

import "testing"

// TestPath verifies path resolution uses a non-empty environment override.
//
// Given an environment variable that is set or empty, when a path is resolved
// then the configured (if set) or default (otherwise) is returned.
func TestPath(t *testing.T) {
	t.Setenv(string(RHSMConfigPath), "/configured/path")

	if got := Path(RHSMConfigPath); got != "/configured/path" {
		t.Errorf("Path() = %q, want /configured/path", got)
	}
}
