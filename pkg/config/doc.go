// Package config provides the resolved heartbeat configuration API.
//
// Get reads the TOML configuration at /etc/rhc/rhc-heartbeat.conf and applies
// supported fallback values from /etc/rhsm/rhsm.conf. The TOML path can be
// overridden with RHC_HEARTBEAT_CONFIG, and the RHSM path can be overridden
// with RHC_HEARTBEAT_RHSM_CONFIG.
//
//	cfg, err := config.Get()
//	if err != nil {
//		return err
//	}
//	endpoint := cfg.OTEL.URI
//	_ = endpoint
package config
