package config

import (
	"fmt"
	"log/slog"
	"os"

	internalconfig "github.com/m-horky/rhc-heartbeat/internal/config"
	"github.com/m-horky/rhc-heartbeat/internal/fs"
)

const (
	configPathEnv = "RHC_HEARTBEAT_CONFIG"
	rhsmPathEnv   = "RHC_HEARTBEAT_RHSM_CONFIG"

	defaultConfigPath = "/etc/rhc/rhc-heartbeat.conf"
	defaultRHSMPath   = "/etc/rhsm/rhsm.conf"
)

// Config is the resolved heartbeat configuration.
type Config = internalconfig.Config

// Endpoint describes the OTLP/HTTP endpoint and its TLS settings.
type Endpoint = internalconfig.Endpoint

// HTTPConfig contains outgoing HTTP transport settings.
type HTTPConfig = internalconfig.HTTPConfig

// Proxy describes the HTTP proxy and its optional credentials.
type Proxy = internalconfig.Proxy

// Get loads heartbeat configuration from environment overrides or the system defaults.
func Get() (Config, error) {
	path := os.Getenv(configPathEnv)
	if path == "" {
		path = defaultConfigPath
		slog.Debug("using default heartbeat configuration", "path", path)
	} else {
		slog.Debug("using configured heartbeat configuration", "path", path)
	}

	rhsmPath := os.Getenv(rhsmPathEnv)
	if rhsmPath == "" {
		rhsmPath = defaultRHSMPath
		slog.Debug("using default RHSM configuration", "path", rhsmPath)
	} else {
		slog.Debug("using configured RHSM configuration", "path", rhsmPath)
	}

	cfg, err := internalconfig.LoadFromPaths(fs.Filesystem{}, path, rhsmPath)
	if err != nil {
		return Config{}, fmt.Errorf("load heartbeat configuration: %w", err)
	}

	return cfg, nil
}
