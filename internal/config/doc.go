// Package config loads native heartbeat configuration and exposes its resolved
// HTTP and API settings. Load starts with embedded defaults, applies an optional
// RHSM fallback, then merges the native configuration file and sorted .conf
// drop-ins. Values from later sources override earlier values.
//
// For example, load RHSM settings first and pass them as a fallback while
// loading native settings:
//
//	filesystem := fs.Filesystem{}
//	rhsmConfig, err := rhsm.Load(filesystem, "/etc/rhsm/rhsm.conf")
//	if err != nil {
//		return err
//	}
//
//	cfg, err := config.Load(
//		filesystem,
//		"/etc/rhc-heartbeat/config.toml",
//		"/etc/rhc-heartbeat/conf.d",
//		&rhsmConfig,
//	)
//	if err != nil {
//		return err
//	}
//	_ = cfg.API.Heartbeat.URI
//	_ = cfg.HTTP.Timeout.Request
package config
