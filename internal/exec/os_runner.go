package exec

import (
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// OSRunner runs commands through the operating system process API.
type OSRunner struct{}

// Compile-time validation that OSRunner implements Runner.
var _ Runner = OSRunner{}

// Run starts the configured command and captures its bounded output.
// A nonzero process exit is reported in Result; errors are reserved for start and other execution failures.
func (OSRunner) Run(cmd Command) (Result, error) {
	if cmd.Path == "" {
		return Result{}, errors.New("run command: executable path is empty")
	}

	if !strings.HasPrefix(cmd.Path, "/") {
		slog.Debug("prefer absolute paths over relative", "binary", cmd.Path)
	}

	// Set an output limit for each process stream.
	stdout := limitedBuffer{limit: maxOutputSize}
	stderr := limitedBuffer{limit: maxOutputSize}

	args := make([]string, 1, len(cmd.Args)+1)
	args[0] = cmd.Path
	args = append(args, cmd.Args...)

	process := exec.Cmd{
		Path:   cmd.Path,
		Args:   args,
		Env:    cmd.Env,
		Dir:    cmd.Dir,
		Stdout: &stdout,
		Stderr: &stderr,
	}

	// Run waits for the process and its output copying to finish.
	runErr := process.Run()

	// Read the buffers only after Run has finished writing to them.
	result := Result{
		Stdout:          stdout.buffer.Bytes(),
		Stderr:          stderr.buffer.Bytes(),
		StdoutTruncated: stdout.truncated,
		StderrTruncated: stderr.truncated,
	}

	if runErr == nil {
		return result, nil
	}

	// A nonzero exit is a command result; other wait failures are execution errors.
	if exitErr, ok := errors.AsType[*exec.ExitError](runErr); ok {
		result.ExitCode = exitErr.ExitCode()

		return result, nil
	}

	return result, fmt.Errorf("run command %q: %w", cmd.Path, runErr)
}
