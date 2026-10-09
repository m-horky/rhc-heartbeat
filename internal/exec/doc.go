// Package exec provides an injectable abstraction for running commands and
// capturing bounded output.
//
// Callers pass a command configuration to a Runner. A nonzero process exit is
// reported in Result.ExitCode, while start and other execution failures are
// returned as errors. Stdout and Stderr are each limited to maxOutputSize bytes.
// Command.Path identifies the executable, while Command.Args contains only
// its additional arguments. Command.Dir independently sets the process working
// directory. A nil Command.Env inherits the current environment; a non-nil
// environment replaces it.
//
//	runner := OSRunner{}
//	result, err := runner.Run(Command{
//		Path: "/usr/bin/dnf",
//		Args: []string{"update"},
//	})
//
//
//	if err != nil {
//		// Handle a lookup, start, or context failure.
//	}
//	if result.ExitCode != 0 {
//		// Handle a nonzero process exit.
//	}
package exec
