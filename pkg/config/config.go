package config

import (
	"fmt"
	"log/slog"
	"os"

	internalconfig "github.com/m-horky/rhc-heartbeat/internal/config"
	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/constants"
)

// Config is the resolved heartbeat configuration.
type Config = internalconfig.Config

// Endpoint describes the heartbeat upload endpoint and its TLS settings.
type Endpoint = internalconfig.Endpoint

// HTTPConfig contains outgoing HTTP transport settings.
type HTTPConfig = internalconfig.HTTPConfig

// Proxy describes the HTTP proxy and its optional credentials.
type Proxy = internalconfig.Proxy

// Get loads heartbeat configuration from environment overrides or the system defaults.
func Get() (Config, error) {
	path := os.Getenv(constants.ConfigPathEnv)
	if path == "" {
		path = constants.DefaultConfigPath
		slog.Debug("using default heartbeat configuration", "path", path)
	} else {
		slog.Debug("using configured heartbeat configuration", "path", path)
	}

	rhsmPath := os.Getenv(constants.RHSMPathEnv)
	if rhsmPath == "" {
		rhsmPath = constants.DefaultRHSMPath
		slog.Debug("using default RHSM configuration", "path", rhsmPath)
	} else {
		slog.Debug("using configured RHSM configuration", "path", rhsmPath)
	}

	cfg, err := internalconfig.LoadFromPaths(fs.Filesystem{}, path, rhsmPath)
	if err != nil {
		return Config{}, fmt.Errorf("load heartbeat configuration: %w", err)
	}

	proxyCredentialLogValue := ""
	if cfg.HTTP.Proxy.User != "" || cfg.HTTP.Proxy.Password != "" {
		proxyCredentialLogValue = "..."
	}

	slog.Debug("resolved heartbeat configuration",
		"heartbeat.uri", cfg.Heartbeat.URI,
		"heartbeat.tls_verify", cfg.Heartbeat.TLSVerify,
		"heartbeat.ca_path", cfg.Heartbeat.CAPath,
		"http.proxy.uri", cfg.HTTP.Proxy.URI,
		"http.proxy.user", proxyCredentialLogValue,
		"http.proxy.password", proxyCredentialLogValue,
	)

	return cfg, nil
}
