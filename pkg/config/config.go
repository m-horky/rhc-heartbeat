package config

import (
	"fmt"
	"log/slog"
	"net/url"

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

// Timeout contains resolved HTTP transport timeouts.
type Timeout = internalconfig.Timeout

// Proxy describes the HTTP proxy, and bypass rules.
type Proxy = internalconfig.Proxy

// Get loads heartbeat configuration from environment overrides or the system defaults.
func Get() (Config, error) {
	path := constants.Path(constants.ConfigPath)
	dropInsDir := constants.Path(constants.ConfigDropInsDir)
	rhsmPath := constants.Path(constants.RHSMConfigPath)

	fallback, err := rhsm.Load(fs.Filesystem{}, rhsmPath)
	if err != nil {
		return Config{}, fmt.Errorf("load RHSM configuration: %w", err)
	}

	loaded, err := internalconfig.Load(fs.Filesystem{}, path, dropInsDir, &fallback)
	if err != nil {
		return Config{}, fmt.Errorf("load heartbeat configuration: %w", err)
	}

	cfg := Config{Heartbeat: loaded.API.Heartbeat, HTTP: loaded.HTTP}

	slog.Debug("resolved configuration",
		"heartbeat.uri", safeURL(cfg.Heartbeat.URI),
		"heartbeat.tls_verify", cfg.Heartbeat.TLSVerify,
		"heartbeat.ca_path", cfg.Heartbeat.CAPath,
		"http.proxy.uri", safeURL(cfg.HTTP.Proxy.URI),
	)

	return cfg, nil
}

// safeURL returns a URL without user information or query secrets for diagnostics.
func safeURL(value url.URL) string {
	value.User = nil
	value.RawQuery = ""
	value.ForceQuery = false
	value.Fragment = ""
	value.RawFragment = ""

	return value.Redacted()
}
