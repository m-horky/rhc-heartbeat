package constants

import "os"

type pathID string

const (
	ConfigPath        pathID = "RHC_HEARTBEAT_CONFIG"
	ConfigDropInsDir  pathID = "RHC_HEARTBEAT_DROPINS"
	PendingCachePath  pathID = "RHC_HEARTBEAT_PENDING_CACHE"
	LockPath          pathID = "RHC_HEARTBEAT_LOCK"
	IdentityCertPath  pathID = "RHC_HEARTBEAT_CONSUMER_CERT"
	IdentityKeyPath   pathID = "RHC_HEARTBEAT_CONSUMER_KEY"
	ProductDir        pathID = "RHC_HEARTBEAT_PRODUCT_DIR"
	DefaultProductDir pathID = "RHC_HEARTBEAT_DEFAULT_PRODUCT_DIR"
	RHSMConfigPath    pathID = "RHC_HEARTBEAT_RHSM_CONFIG"
	RHSMBinPath       pathID = "RHC_HEARTBEAT_RHSM_BIN"
)

var paths = map[pathID]string{ //nolint:gochecknoglobals
	ConfigPath:        "/etc/rhc/rhc-heartbeat.conf",
	ConfigDropInsDir:  "/etc/rhc/rhc-heartbeat.conf.d/",
	PendingCachePath:  "/var/lib/rhc/heartbeat.jsonl",
	LockPath:          "/run/lock/rhc-heartbeat.lock",
	IdentityCertPath:  "/etc/pki/consumer/cert.pem",
	IdentityKeyPath:   "/etc/pki/consumer/key.pem",
	ProductDir:        "/etc/pki/product",
	DefaultProductDir: "/etc/pki/product-default",
	RHSMConfigPath:    "/etc/rhsm/rhsm.conf",
	RHSMBinPath:       "/usr/libexec/rhc/heartbeat-rhsm",
}

func Path(name pathID) string {
	if path := os.Getenv(string(name)); path != "" {
		return path
	}

	return paths[name]
}
