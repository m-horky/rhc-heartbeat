// Package profile implements the profile subcommand of heartbeat.
//
// Call Run with the process context and command arguments to collect and print
// a periodic heartbeat as JSON:
//
//	if err := profile.Run(ctx, nil); err != nil {
//		return err
//	}
package profile
