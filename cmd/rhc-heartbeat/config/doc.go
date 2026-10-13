// Package config implements the config subcommand of heartbeat.
//
// Call Run with the process context and command arguments to print the resolved,
// credential-redacted configuration as JSON:
//
//	if err := config.Run(ctx, nil); err != nil {
//		return err
//	}
package config
