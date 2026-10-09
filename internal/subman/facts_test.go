package subman

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/m-horky/rhc-heartbeat/internal/exec"
)

// TestParseFactsParsesValues verifies facts preserve values containing separators.
//
// Given subscription-manager output with key/value facts, when parsed,
// then each fact is trimmed and colons in values are retained.
func TestParseFactsParsesValues(t *testing.T) {
	t.Parallel()

	got, err := parseFactsOutput([]byte(
		"  lscpu.cpu(s): 4  \naws_instance_id: i-123\nendpoint: https://example.test:443\n",
	))
	if err != nil {
		t.Fatalf("parseFactsOutput() error = %v", err)
	}

	want := map[string]string{
		"lscpu.cpu(s)":    "4",
		"aws_instance_id": "i-123",
		"endpoint":        "https://example.test:443",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parseFactsOutput() = %#v, want %#v", got, want)
	}
}

// TestParseFactsRejectsMalformedAndDuplicateLines verifies malformed fact output is not silently accepted.
//
// Given malformed or duplicate fact lines, when parsed, then an error identifies the line.
func TestParseFactsRejectsMalformedAndDuplicateLines(t *testing.T) {
	t.Parallel()

	for _, output := range []string{
		"valid: yes\nnot a fact\n",
		"duplicate: first\nduplicate: second\n",
	} {
		if _, err := parseFactsOutput([]byte(output)); err == nil {
			t.Errorf("parseFactsOutput(%q) error = nil, want an error", output)
		}
	}
}

// TestReadRunsAndParsesSubscriptionManager verifies Read invokes the expected command and parses its result.
//
// Given a successful subscription-manager runner, when facts are read, then its command output is returned as facts.
func TestReadRunsAndParsesSubscriptionManager(t *testing.T) {
	t.Parallel()

	runner := runnerFunc(func(command exec.Command) (exec.Result, error) {
		if command.Path != "/usr/sbin/subscription-manager" {
			t.Errorf("command path = %q, want %q", command.Path, "/usr/sbin/subscription-manager")
		}

		if !reflect.DeepEqual(command.Args, []string{"facts", "--list"}) {
			t.Errorf("command args = %#v, want [facts --list]", command.Args)
		}

		return exec.Result{Stdout: []byte("aws_instance_id: i-123\n")}, nil
	})

	got, err := Read(runner)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if got["aws_instance_id"] != "i-123" {
		t.Errorf("Read() facts = %#v, want aws_instance_id i-123", got)
	}
}

// TestReadRejectsFailedOrTruncatedCommandOutput verifies incomplete command results are treated as failures.
//
// Given a failed or truncated command result, when facts are read, then Read returns an explanatory error.
func TestReadRejectsFailedOrTruncatedCommandOutput(t *testing.T) {
	t.Parallel()

	for _, result := range []exec.Result{
		{ExitCode: 1, Stderr: []byte("denied")},
		{Stdout: []byte("fact: value"), StdoutTruncated: true},
	} {
		if _, err := Read(runnerFunc(func(exec.Command) (exec.Result, error) { return result, nil })); err == nil {
			t.Errorf("Read() error = nil for result %#v, want an error", result)
		}
	}
}

// TestReadWrapsRunnerError verifies process execution errors retain their cause.
//
// Given the command runner fails, when facts are read, then its error remains discoverable.
func TestReadWrapsRunnerError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("runner failed")

	_, err := Read(runnerFunc(func(exec.Command) (exec.Result, error) { return exec.Result{}, wantErr }))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Read() error = %v, want wrapped %v", err, wantErr)
	}

	if !strings.Contains(err.Error(), "subscription-manager facts") {
		t.Errorf("Read() error = %v, want command context", err)
	}
}

type runnerFunc func(exec.Command) (exec.Result, error)

// Run executes a test-defined command behavior.
func (run runnerFunc) Run(command exec.Command) (exec.Result, error) {
	return run(command)
}
