// Package command provides an injectable interface for running external programs.
package command

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// Result contains the captured standard output and standard error of a command.
type Result struct {
	Stdout []byte
	Stderr []byte
}

// Runner executes a program with the provided arguments and context.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (Result, error)
}

// OSRunner executes explicitly approved programs using the operating system's process facilities.
type OSRunner struct{}

// Compile-time validation that OSRunner implements Runner.
var _ Runner = OSRunner{}

// Run executes name with args, capturing its standard output and standard error.
func (OSRunner) Run(ctx context.Context, name string, args ...string) (Result, error) {
	var cmd *exec.Cmd

	switch {
	case name == "chronyc" && len(args) == 2 && args[0] == "-c" && args[1] == "tracking":
		cmd = exec.CommandContext(ctx, "chronyc", "-c", "tracking")
	default:
		return Result{}, fmt.Errorf("unsupported command invocation %q %q", name, args)
	}

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := Result{
		Stdout: stdout.Bytes(),
		Stderr: stderr.Bytes(),
	}
	if err != nil {
		return result, fmt.Errorf("execute %s: %w", name, err)
	}

	return result, nil
}
