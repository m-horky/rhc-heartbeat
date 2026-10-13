package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// TestPrintHelpListsCommandsAndFlags verifies generated help describes all commands and flags.
//
// Given the command definitions, when help is generated, then the available commands and flags are listed.
func TestPrintHelpListsCommandsAndFlags(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	if err := printHelp(&output); err != nil {
		t.Fatalf("printHelp() error = %v", err)
	}

	for _, expected := range []string{"Usage: heartbeat", "run", "config", "profile", "-kind", "ping"} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("generated help = %q, want it to contain %q", output.String(), expected)
		}
	}
}

// TestRunRequiresExplicitSubcommand verifies command-line options do not implicitly start heartbeat processing.
//
// Given run options without the run subcommand, when the command is dispatched,
// then it returns an unknown-command error.
func TestRunRequiresExplicitSubcommand(t *testing.T) {
	t.Parallel()

	err := run(context.Background(), []string{"--kind=ping"})
	if err == nil || !strings.Contains(err.Error(), "unknown subcommand") {
		t.Fatalf("run() error = %v, want an unknown-subcommand error", err)
	}
}

// TestInspectionSubcommandsRejectArguments verifies config and profile accept no positional arguments.
//
// Given an inspection subcommand with extra arguments, when command dispatch runs, then it returns a validation error.
func TestInspectionSubcommandsRejectArguments(t *testing.T) {
	t.Parallel()

	for _, subcommand := range []string{"config", "profile"} {
		t.Run(subcommand, func(t *testing.T) {
			t.Parallel()

			err := run(context.Background(), []string{subcommand, "unexpected"})
			if err == nil || !strings.Contains(err.Error(), "does not accept arguments") {
				t.Fatalf("run() error = %v, want an argument validation error", err)
			}
		})
	}
}
