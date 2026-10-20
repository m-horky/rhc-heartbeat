// Command heartbeat-rhsm execs subscription-manager(8) in the rhsmcertd_t
// SELinux domain. rhsmcertd_t can only be entered through a file holding
// an entrypoint permission, and /usr/sbin/subscription-manager is generic
// bin_t. This wrapper carries the rhc_heartbeat_rhsm_exec_t label and
// transitions into rhsmcertd_t, which may execute bin_t with no further
// transitions.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

const subscriptionManager = "/usr/sbin/subscription-manager"

// main execs subscription-manager with the same arguments, forwarding file
// descriptors and the exit status, and reports setup failures on stderr.
func main() {
	args := make([]string, 1, len(os.Args))
	args[0] = subscriptionManager
	args = append(args, os.Args[1:]...)

	cmd := exec.Cmd{
		Path:   subscriptionManager,
		Args:   args,
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := cmd.Run(); err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			os.Exit(exitErr.ExitCode())
		}

		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}
