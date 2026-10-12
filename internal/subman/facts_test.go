package subman

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/m-horky/rhc-heartbeat/internal/exec"
)

// TestParse verifies facts preserve values containing separators.
//
// Given subscription-manager output with key/value facts, when parsed,
// then each fact is trimmed and colons in values are retained.
func TestParse(t *testing.T) {
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
	if !cmp.Equal(got, want) {
		t.Errorf("parseFactsOutput() = %#v, want %#v", got, want)
	}
}

// TestExitCode verifies incomplete command results are treated as failures.
//
// Given a failed or truncated command result, when facts are read, then GetFacts returns an explanatory error.
func TestExitCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result exec.Result
		err    error
	}{
		{
			name:   "exec fails",
			result: exec.Result{},
			err:    errors.New("command not found"),
		},
		{
			name:   "exit code",
			result: exec.Result{ExitCode: 1},
			err:    nil,
		},
		{
			name:   "truncated",
			result: exec.Result{Stdout: []byte(""), StdoutTruncated: true},
			err:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			r := runner{result: test.result, err: test.err}

			if _, err := GetFacts(r); err == nil {
				t.Errorf("Read() error = nil for result %#v, want an error", test.result)
			}
		})
	}
}

type runner struct {
	result exec.Result
	err    error
}

func (r runner) Run(_ exec.Command) (exec.Result, error) {
	return r.result, r.err
}
