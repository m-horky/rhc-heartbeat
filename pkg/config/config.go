package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"

	internalconfig "github.com/m-horky/rhc-heartbeat/internal/config"
	"github.com/m-horky/rhc-heartbeat/internal/config/rhsm"
	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/constants"
)

// Config is the resolved heartbeat configuration exposed to applications.
type Config struct {
	Heartbeat Endpoint
	HTTP      HTTPConfig
}

// Endpoint describes the heartbeat upload endpoint and its TLS settings.
type Endpoint = internalconfig.Endpoint

// HTTPConfig contains outgoing HTTP transport settings.
type HTTPConfig = internalconfig.HTTP

// Proxy describes the HTTP proxy, its credentials, and bypass rules.
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

	fallback, err := rhsm.Load(fs.Filesystem{}, rhsmPath)
	if err != nil {
		return Config{}, fmt.Errorf("load RHSM configuration: %w", err)
	}

	loaded, err := internalconfig.Load(fs.Filesystem{}, path, constants.DefaultDropInDir, &fallback)
	if err != nil {
		return Config{}, fmt.Errorf("load heartbeat configuration: %w", err)
	}

	cfg := Config{Heartbeat: loaded.API.Heartbeat, HTTP: loaded.HTTP}

	proxyCredentialLogValue := ""
	if cfg.HTTP.Proxy.Username != "" || cfg.HTTP.Proxy.Password != "" {
		proxyCredentialLogValue = "..."
	}

	slog.Debug("resolved heartbeat configuration",
		"heartbeat.uri", cfg.Heartbeat.URI.String(),
		"heartbeat.tls_verify", cfg.Heartbeat.TLSVerify,
		"heartbeat.ca_path", cfg.Heartbeat.CAPath,
		"http.proxy.uri", safeURL(cfg.HTTP.Proxy.URI),
		"http.proxy.username", proxyCredentialLogValue,
		"http.proxy.password", proxyCredentialLogValue,
	)

	return cfg, nil
}

// safeURL returns a URL without user information for diagnostics.
func safeURL(value url.URL) string {
	value.User = nil

	return value.String()
}
