package config

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Config contains resolved native configuration.
type Config struct {
	HTTP HTTP
	API  API
}

// HTTP contains resolved outgoing HTTP settings.
type HTTP struct {
	Timeout Timeout
	Proxy   Proxy
}

// Timeout contains resolved transport timeouts.
type Timeout struct {
	Connect time.Duration
	Request time.Duration
	Idle    time.Duration
}

// Proxy contains a proxy URL (including credentials) and bypass rules.
type Proxy struct {
	URI     url.URL
	NoProxy string
}

// API contains remote endpoints.
type API struct {
	Heartbeat Endpoint
}

// Endpoint contains an endpoint URL and TLS settings.
type Endpoint struct {
	URI       url.URL
	TLSVerify bool
	CAPath    string
}

// resolve converts merged native settings into complete resolved sections.
func (partial *dtoConfig) resolve() (Config, error) {
	endpoint, err := resolveEndpoint(partial.API.Heartbeat)
	if err != nil {
		return Config{}, err
	}

	proxy, err := resolveProxy(partial.HTTP.Proxy)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTP: HTTP{
			Timeout: Timeout{
				Connect: time.Duration(*partial.HTTP.Timeout.Connect) * time.Second,
				Request: time.Duration(*partial.HTTP.Timeout.Request) * time.Second,
				Idle:    time.Duration(*partial.HTTP.Timeout.Idle) * time.Second,
			},
			Proxy: proxy,
		},
		API: API{Heartbeat: endpoint},
	}, nil
}

// resolveEndpoint parses the configured endpoint and resolves its TLS settings.
func resolveEndpoint(partial dtoEndpoint) (Endpoint, error) {
	if partial.URI == nil || *partial.URI == "" || partial.TLSVerify == nil || partial.CAPath == nil {
		return Endpoint{}, errors.New("resolve api.heartbeat: required setting is empty or missing")
	}

	err := isValidURI(*partial.URI)
	if err != nil {
		return Endpoint{}, fmt.Errorf("resolve api.heartbeat.uri: %w", err)
	}

	uri, err := url.Parse(*partial.URI)
	if err != nil {
		return Endpoint{}, fmt.Errorf("resolve api.heartbeat.uri: %w", err)
	}

	return Endpoint{URI: *uri, TLSVerify: *partial.TLSVerify, CAPath: *partial.CAPath}, nil
}

// resolveProxy parses the configured proxy URI and resolves optional credentials and bypass rules.
func resolveProxy(partial dtoProxy) (Proxy, error) {
	if partial.URI == nil || *partial.URI == "" {
		return Proxy{}, nil
	}

	if err := isValidURI(*partial.URI); err != nil {
		return Proxy{}, fmt.Errorf("resolve http.proxy.uri: %w", err)
	}

	uri, err := url.Parse(*partial.URI)
	if err != nil {
		return Proxy{}, fmt.Errorf("resolve http.proxy.uri: %w", err)
	}

	return Proxy{
		URI:     *uri,
		NoProxy: *partial.NoProxy,
	}, nil
}
