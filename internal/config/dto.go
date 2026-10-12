package config

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"strings"
	"time"
)

// dtoConfig contains one partial native configuration layer.
type dtoConfig struct {
	HTTP dtoHTTP `toml:"http"`
	API  dtoAPI  `toml:"api"`
}

// dtoHTTP contains partial outgoing HTTP settings.
type dtoHTTP struct {
	Timeout dtoTimeout `toml:"timeout"`
	Proxy   dtoProxy   `toml:"proxy"`
}

// dtoTimeout contains optional timeout values measured in seconds.
type dtoTimeout struct {
	Connect *int64 `toml:"connect"`
	Request *int64 `toml:"request"`
	Idle    *int64 `toml:"idle"`
}

// dtoProxy contains optional proxy settings, including bypass rules.
type dtoProxy struct {
	URI     *string `toml:"uri"`
	NoProxy *string `toml:"no-proxy"`
}

// dtoAPI contains partial API endpoint settings.
type dtoAPI struct {
	Heartbeat dtoEndpoint `toml:"heartbeat"`
}

// dtoEndpoint contains optional endpoint and TLS settings.
type dtoEndpoint struct {
	URI       *string `toml:"uri"`
	TLSVerify *bool   `toml:"tls-verify"`
	CAPath    *string `toml:"ca-path"`
}

// update validates and merges the supplied partial configuration into the current dtoConfig.
func (partial *dtoConfig) update(next *dtoConfig) error {
	// TODO Should we prefer a new object over mutation?
	if next == nil {
		return nil
	}

	if err := partial.HTTP.Timeout.update(next.HTTP.Timeout); err != nil {
		return fmt.Errorf("invalid http.timeout: %w", err)
	}

	if err := partial.HTTP.Proxy.update(next.HTTP.Proxy); err != nil {
		return fmt.Errorf("invalid http.proxy: %w", err)
	}

	if err := partial.API.Heartbeat.update(next.API.Heartbeat); err != nil {
		return fmt.Errorf("invalid api.heartbeat: %w", err)
	}

	return nil
}

// update validates and merges HTTP timeout preferences.
func (partial *dtoTimeout) update(next dtoTimeout) error {
	if next.Connect != nil {
		if ok := isTimeDuration(*next.Connect); !ok {
			return fmt.Errorf("invalid connect duration: %d", *next.Connect)
		}

		partial.Connect = next.Connect
	}

	if next.Request != nil {
		if ok := isTimeDuration(*next.Request); !ok {
			return fmt.Errorf("invalid request duration: %d", next.Request)
		}

		partial.Request = next.Request
	}

	if next.Idle != nil {
		if ok := isTimeDuration(*next.Idle); !ok {
			return fmt.Errorf("invalid idle duration: %d", *next.Idle)
		}

		partial.Idle = next.Idle
	}

	return nil
}

// update validates and merges proxy URLs, credentials, and bypass rules.
func (partial *dtoProxy) update(next dtoProxy) error {
	if next.URI == nil || *next.URI == "" {
		return nil
	}

	if err := isValidURI(*next.URI); err != nil {
		return err
	}

	partial.URI = new(*next.URI)

	if next.NoProxy != nil {
		partial.NoProxy = new(*next.NoProxy)
	}

	return nil
}

// update validates and merges endpoint URL and TLS settings.
func (partial *dtoEndpoint) update(next dtoEndpoint) error {
	if next.URI != nil {
		if err := isValidURI(*next.URI); err != nil {
			return err
		}

		partial.URI = new(*next.URI)
	}

	if next.TLSVerify != nil {
		partial.TLSVerify = new(*next.TLSVerify)
	}

	if next.CAPath != nil {
		partial.CAPath = new(strings.TrimSpace(*next.CAPath))
	}

	return nil
}

// isValidURI validates a supplied URI.
func isValidURI(value string) error {
	if s := strings.TrimSpace(value); s == "" {
		return errors.New("no content")
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("invalid URI: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https")
	}

	if parsed.Hostname() == "" {
		return fmt.Errorf("no hostname")
	}

	if parsed.Scheme == "http" {
		slog.Warn("configuration contains insecure HTTP")
	}

	return nil
}

// isTimeDuration validates the input can be represented as a time.Duration.
func isTimeDuration(seconds int64) bool {
	return seconds >= 0 && seconds <= math.MaxInt64/int64(time.Second)
}
