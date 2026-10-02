package constants

import "testing"

// TestPathFromEnvUsesOverrideOrDefault verifies path resolution uses a non-empty environment override.
//
// Given an environment variable that is set or empty
// When a path is resolved
// Then the configured path is returned, or the default is used when no override is provided.
func TestPathFromEnvUsesOverrideOrDefault(t *testing.T) {
	t.Setenv("RHC_HEARTBEAT_TEST_PATH", "/configured/path")

	if got := PathFromEnv("RHC_HEARTBEAT_TEST_PATH", "/default/path"); got != "/configured/path" {
		t.Errorf("PathFromEnv() = %q, want configured path", got)
	}

	t.Setenv("RHC_HEARTBEAT_TEST_PATH", "")

	if got := PathFromEnv("RHC_HEARTBEAT_TEST_PATH", "/default/path"); got != "/default/path" {
		t.Errorf("PathFromEnv() = %q, want default path", got)
	}
}
