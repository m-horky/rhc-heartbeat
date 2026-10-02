// Package config loads heartbeat settings and compatible values from rhsm.conf.
package config

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"net/url"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	DefaultPath = "/etc/rhc/rhc-heartbeat.conf"
	RHSMPath    = "/etc/rhsm/rhsm.conf"
)

// Config is the resolved heartbeat configuration.
type Config struct {
	OTEL Endpoint   `toml:"otel"`
	HTTP HTTPConfig `toml:"http"`
}

// HTTPConfig contains transport settings.
type HTTPConfig struct {
	Proxy Proxy `toml:"proxy"`
}

// Endpoint is the OTLP/HTTP logs endpoint and its TLS settings.
type Endpoint struct {
	URI       string `toml:"uri"`
	TLSVerify bool   `toml:"tls-verify"`
	CAPath    string `toml:"ca-path"`
}

// Proxy contains HTTP proxy settings shared by outgoing requests.
type Proxy struct {
	URI      string `toml:"uri"`
	User     string `toml:"user"`
	Password string `toml:"password"`
}

type partialConfig struct {
	OTEL *partialEndpoint `toml:"otel"`
	HTTP *partialHTTP     `toml:"http"`
}

type partialHTTP struct {
	Proxy *partialProxy `toml:"proxy"`
}

type partialEndpoint struct {
	URI       *string `toml:"uri"`
	TLSVerify *bool   `toml:"tls-verify"`
	CAPath    *string `toml:"ca-path"`
}

type partialProxy struct {
	URI      *string `toml:"uri"`
	User     *string `toml:"user"`
	Password *string `toml:"password"`
}

// Load reads the application configuration and rhsm.conf from their system paths.
func Load() (Config, error) { return LoadFromPaths(DefaultPath, RHSMPath) }

// LoadFromPaths resolves the application TOML file over rhsm.conf-derived defaults.
// Missing files are allowed; errors reading or parsing present files are returned.
func LoadFromPaths(configPath, rhsmPath string) (Config, error) {
	cfg := Config{OTEL: Endpoint{TLSVerify: true}}

	legacy, err := loadRHSM(rhsmPath)
	if err != nil {
		return Config{}, err
	}

	cfg = cfg.applyRHSM(legacy)

	//nolint:gosec // Load uses fixed system paths; LoadFromPaths supports controlled injection.
	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, iofs.ErrNotExist) {
			return cfg, nil
		}

		return Config{}, fmt.Errorf("read configuration %s: %w", configPath, err)
	}

	var override partialConfig
	if _, err := toml.Decode(string(data), &override); err != nil {
		return Config{}, fmt.Errorf("decode configuration %s: invalid TOML syntax", configPath)
	}

	if err := cfg.apply(override); err != nil {
		return Config{}, fmt.Errorf("validate configuration %s: %w", configPath, err)
	}

	return cfg, nil
}

// applyRHSM applies non-empty legacy settings as configuration fallbacks.
func (cfg Config) applyRHSM(legacy rhsmSettings) Config {
	if legacy.CandlepinURI != "" {
		cfg.OTEL.URI = legacy.OTELURI
	}

	if legacy.InsecurePresent {
		cfg.OTEL.TLSVerify = !legacy.Insecure
	}

	if legacy.CAPath != "" {
		cfg.OTEL.CAPath = legacy.CAPath
	}

	if legacy.Proxy.URI != "" || legacy.Proxy.User != "" || legacy.Proxy.Password != "" {
		cfg.HTTP.Proxy = legacy.Proxy
	}

	return cfg
}

// apply applies explicitly supplied TOML values over resolved fallback settings.
func (cfg *Config) apply(p partialConfig) error {
	if p.OTEL != nil {
		if err := cfg.applyEndpoint(*p.OTEL); err != nil {
			return err
		}
	}

	if p.HTTP != nil && p.HTTP.Proxy != nil {
		if err := cfg.applyProxy(*p.HTTP.Proxy); err != nil {
			return err
		}
	}

	return nil
}

// applyEndpoint validates and applies an OTEL endpoint override.
func (cfg *Config) applyEndpoint(p partialEndpoint) error {
	if p.URI != nil {
		uri, err := applyURI(*p.URI, validateEndpointURI)
		if err != nil {
			return fmt.Errorf("otel.uri: %w", err)
		}

		cfg.OTEL.URI = uri
	}

	if p.TLSVerify != nil {
		cfg.OTEL.TLSVerify = *p.TLSVerify
	}

	if p.CAPath != nil {
		cfg.OTEL.CAPath = strings.TrimSpace(*p.CAPath)
	}

	return nil
}

// applyProxy validates and applies HTTP proxy overrides.
func (cfg *Config) applyProxy(p partialProxy) error {
	if p.URI != nil {
		uri, err := applyURI(*p.URI, validateProxyURI)
		if err != nil {
			return fmt.Errorf("http.proxy.uri: %w", err)
		}

		cfg.HTTP.Proxy.URI = uri
	}

	if p.User != nil {
		cfg.HTTP.Proxy.User = strings.TrimSpace(*p.User)
	}

	if p.Password != nil {
		cfg.HTTP.Proxy.Password = *p.Password
	}

	return nil
}

// applyURI accepts an empty override or validates a non-empty URI.
func applyURI(value string, validate func(string) (string, error)) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	return validate(value)
}

// validateEndpointURI accepts HTTP(S) endpoint URIs without credentials or request data.
func validateEndpointURI(value string) (string, error) {
	value = strings.TrimSpace(value)

	u, err := url.ParseRequestURI(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" ||
		u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("invalid HTTP(S) endpoint URI")
	}

	return value, nil
}

// validateProxyURI accepts HTTP(S) proxy URIs without embedded credentials or request data.
func validateProxyURI(value string) (string, error) {
	u, err := url.ParseRequestURI(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" ||
		u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("invalid HTTP(S) proxy URI")
	}

	return value, nil
}
