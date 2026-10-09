package subman

import (
	"errors"
	"fmt"
	"strings"

	"github.com/m-horky/rhc-heartbeat/internal/exec"
)

// Read executes subscription-manager facts --list and parses its output.
func Read(runner exec.Runner) (map[string]string, error) {
	result, err := runner.Run(exec.Command{
		Path: "/usr/sbin/subscription-manager",
		Args: []string{"facts", "--list"},
	})
	if err != nil {
		return nil, fmt.Errorf("run subscription-manager facts: %w", err)
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("run subscription-manager facts: exit status %d: %s",
			result.ExitCode, strings.TrimSpace(string(result.Stderr)))
	}

	if result.StdoutTruncated {
		return nil, errors.New("parse subscription-manager facts: output was truncated")
	}

	facts, err := parseFactsOutput(result.Stdout)
	if err != nil {
		return nil, fmt.Errorf("parse subscription-manager facts: %w", err)
	}

	return facts, nil
}

// parseFactsOutput parses subscription-manager's key: value output, splitting each
// non-empty line at its first colon so values may contain colons.
func parseFactsOutput(output []byte) (map[string]string, error) {
	facts := make(map[string]string)
	lines := strings.Split(string(output), "\n")

	for lineNumber, line := range lines {
		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}

		key, value, found := strings.Cut(text, ":")

		key = strings.TrimSpace(key)
		if !found || key == "" {
			return nil, fmt.Errorf("line %d is not a fact", lineNumber+1)
		}

		if _, exists := facts[key]; exists {
			return nil, fmt.Errorf("line %d repeats fact %q", lineNumber+1, key)
		}

		facts[key] = strings.TrimSpace(value)
	}

	return facts, nil
}
