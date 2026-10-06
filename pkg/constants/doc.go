// Package constants exposes the system paths and environment variable names used by rhc-heartbeat.
//
// The default paths cover the application configuration, legacy RHSM configuration, pending-heartbeat
// cache, process lock, and Remote Write client certificate and key. The corresponding path environment
// variables are exported alongside their defaults where overrides are supported.
//
// PathFromEnv returns a non-empty environment override or the supplied default path:
//
//	path := constants.PathFromEnv(constants.PendingCachePathEnv, constants.DefaultPendingCachePath)
//	_ = path
package constants
