// Package constants defines the system paths and environment variables used by rhc-heartbeat.
package constants

import "os"

const (
	// ConfigPathEnv selects the heartbeat configuration file.
	ConfigPathEnv = "RHC_HEARTBEAT_CONFIG"
	// DefaultConfigPath is the default heartbeat configuration file.
	DefaultConfigPath = "/etc/rhc/rhc-heartbeat.conf"

	// RHSMPathEnv selects the legacy RHSM configuration file.
	RHSMPathEnv = "RHC_HEARTBEAT_RHSM_CONFIG"
	// DefaultRHSMPath is the default legacy RHSM configuration file.
	DefaultRHSMPath = "/etc/rhsm/rhsm.conf"

	// PendingCachePathEnv selects the persistent cache file for undelivered heartbeats.
	PendingCachePathEnv = "RHC_HEARTBEAT_PENDING_CACHE_PATH"
	// DefaultPendingCachePath is the default persistent cache file for undelivered heartbeats.
	DefaultPendingCachePath = "/var/lib/rhc/heartbeat.jsonl"

	// DefaultProcessLockPath is the global runtime lock file for rhc-heartbeat.
	DefaultProcessLockPath = "/run/lock/rhc-heartbeat.lock"

	// ClientCertificatePathEnv selects the client certificate used for Remote Write mutual TLS.
	ClientCertificatePathEnv = "RHC_HEARTBEAT_CLIENT_CERT_PATH"
	// DefaultClientCertificatePath is the default client certificate used for Remote Write mutual TLS.
	DefaultClientCertificatePath = "/etc/pki/consumer/cert.pem"

	// ClientKeyPathEnv selects the client key used for Remote Write mutual TLS.
	ClientKeyPathEnv = "RHC_HEARTBEAT_CLIENT_KEY_PATH"
	// DefaultClientKeyPath is the default client key used for Remote Write mutual TLS.
	DefaultClientKeyPath = "/etc/pki/consumer/key.pem"
)

// PathFromEnv returns the environment variable's path, or defaultPath when it is unset or empty.
func PathFromEnv(environmentVariable, defaultPath string) string {
	if path := os.Getenv(environmentVariable); path != "" {
		return path
	}

	return defaultPath
}
