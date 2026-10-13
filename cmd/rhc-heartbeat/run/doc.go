// Package run implements the run subcommand of heartbeat.
//
// Call Run with a process context and the command-line arguments to collect and
// upload a heartbeat:
//
//	if err := run.Run(ctx, []string{"--kind=ping"}); err != nil {
//		return err
//	}
package run
