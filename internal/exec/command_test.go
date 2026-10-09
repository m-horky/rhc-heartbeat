package exec

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestOSRunnerCapturesOutput verifies that successful commands capture both output streams.
//
// Given a command that writes to stdout and stderr, when Run executes it, then both streams are returned.
func TestOSRunnerCapturesOutput(t *testing.T) {
	t.Parallel()

	result, err := OSRunner{}.Run(Command{
		Path: "/bin/sh",
		Args: []string{"-c", "printf stdout; printf stderr >&2"},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !bytes.Equal(result.Stdout, []byte("stdout")) {
		t.Errorf("Run() stdout = %q, want %q", result.Stdout, "stdout")
	}

	if !bytes.Equal(result.Stderr, []byte("stderr")) {
		t.Errorf("Run() stderr = %q, want %q", result.Stderr, "stderr")
	}

	if result.ExitCode != 0 {
		t.Errorf("Run() exit code = %d, want 0", result.ExitCode)
	}
}

// TestOSRunnerReportsNonzeroExit verifies that a nonzero process exit is reported in Result.
//
// Given a command that exits unsuccessfully, when Run executes it, then the result contains its exit code and output.
func TestOSRunnerReportsNonzeroExit(t *testing.T) {
	t.Parallel()

	result, err := OSRunner{}.Run(Command{
		Path: "/bin/sh",
		Args: []string{"-c", "printf failure >&2; exit 7"},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if result.ExitCode != 7 {
		t.Errorf("Run() exit code = %d, want 7", result.ExitCode)
	}

	if !bytes.Equal(result.Stderr, []byte("failure")) {
		t.Errorf("Run() stderr = %q, want %q", result.Stderr, "failure")
	}
}

// TestOSRunnerUsesEnvironmentAndDirectory verifies that configured process environment and directory are applied.
//
// Given a command with an explicit environment and working directory, when Run executes it,
// then it observes both values.
func TestOSRunnerUsesEnvironmentAndDirectory(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	result, err := OSRunner{}.Run(Command{
		Path: "/bin/sh",
		Args: []string{"-c", "printf '%s\\n' \"$CACHE2_TEST_VALUE\"; pwd"},
		Env:  []string{"CACHE2_TEST_VALUE=provided"},
		Dir:  directory,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	want := []byte("provided\n" + filepath.Clean(directory) + "\n")
	if !bytes.Equal(result.Stdout, want) {
		t.Errorf("Run() stdout = %q, want %q", result.Stdout, want)
	}
}

// TestOSRunnerReturnsStartError verifies that a command that cannot be started returns a separate error.
//
// Given a command with a missing executable, when Run starts it, then Run returns an error rather than an exit result.
func TestOSRunnerReturnsStartError(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing")

	result, err := OSRunner{}.Run(Command{Path: missing})
	if err == nil {
		t.Fatal("Run() error = nil, want start error")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Run() error = %v, want wrapped not-exist error", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Run() exit code = %d, want 0 for a start error", result.ExitCode)
	}
}

// TestOSRunnerRejectsEmptyPath verifies that Run rejects a command without an executable path.
//
// Given a command with no executable path, when Run executes it, then Run returns an error.
func TestOSRunnerRejectsEmptyPath(t *testing.T) {
	t.Parallel()

	if _, err := (OSRunner{}).Run(Command{}); err == nil {
		t.Fatal("Run() error = nil, want missing executable path error")
	}
}

// TestOSRunnerRunsConfiguredExecutable verifies that Run executes the configured executable path.
//
// Given an executable path, when Run executes that path, then the executable runs.
func TestOSRunnerRunsConfiguredExecutable(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()

	executable := filepath.Join(directory, "exec-test-command")
	if err := os.Symlink("/bin/sh", executable); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	result, err := OSRunner{}.Run(Command{
		Path: executable,
		Args: []string{"-c", "printf found"},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !bytes.Equal(result.Stdout, []byte("found")) {
		t.Errorf("Run() stdout = %q, want %q", result.Stdout, "found")
	}
}

// TestOSRunnerBoundsCapturedOutput verifies that stdout capture does not exceed its limit.
//
// Given a command that writes more than the output limit, when Run executes it,
// then stdout is bounded and marked as truncated.
func TestOSRunnerBoundsCapturedOutput(t *testing.T) {
	t.Parallel()

	result, err := OSRunner{}.Run(Command{
		Path: "/bin/sh",
		Args: []string{"-c", "printf '%1048577s' ''"},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if len(result.Stdout) != maxOutputSize {
		t.Errorf("captured stdout size = %d, want %d", len(result.Stdout), maxOutputSize)
	}

	if !result.StdoutTruncated {
		t.Error("Run() stdout truncation = false, want true")
	}
}

// TestRunnerReturnsFakeResult verifies that callers can inject a runner and receive a predetermined result.
//
// Given a fake runner and command specification, when a caller runs the command,
// then it receives the fake result and the runner receives the specification.
func TestRunnerReturnsFakeResult(t *testing.T) {
	t.Parallel()

	wantSpec := Command{
		Path: "dnf",
		Args: []string{"update", "--refresh"},
		Env:  []string{"TEST_MODE=1"},
		Dir:  "/var/cache",
	}
	wantResult := Result{Stdout: []byte("updated"), Stderr: []byte("warning"), ExitCode: 3}
	runner := &fakeRunner{result: wantResult}

	var commandRunner Runner = runner

	got, err := commandRunner.Run(wantSpec)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !reflect.DeepEqual(runner.spec, wantSpec) {
		t.Errorf("Run() spec = %#v, want %#v", runner.spec, wantSpec)
	}

	if !reflect.DeepEqual(got, wantResult) {
		t.Errorf("Run() result = %#v, want %#v", got, wantResult)
	}
}

type fakeRunner struct {
	spec   Command
	result Result
}

// Run records the requested command and returns the configured result.
func (runner *fakeRunner) Run(spec Command) (Result, error) {
	runner.spec = spec

	return runner.result, nil
}
