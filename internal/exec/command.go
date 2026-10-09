package exec

// Runner executes a command from its configuration and returns its captured result.
type Runner interface {
	Run(cmd Command) (Result, error)
}

// Command contains the executable path, its arguments, environment, and working directory.
// A nil Env inherits the current process environment; a non-nil Env replaces it.
type Command struct {
	Path string   // path to the binary (prefer absolute)
	Args []string // arguments passed to the binary
	Env  []string // execution environment; leave nil to inherit
	Dir  string   // execution directory; leave empty for CWD
}

// Result contains captured command output, the process exit code, and truncation status.
// Stdout and Stderr are each limited to maxOutputSize bytes.
type Result struct {
	Stdout          []byte
	Stderr          []byte
	ExitCode        int
	StdoutTruncated bool
	StderrTruncated bool
}
