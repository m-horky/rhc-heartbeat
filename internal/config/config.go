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

// Proxy contains a proxy URL, credentials, and bypass rules. NoProxy rules match
// hostnames and their subdomains on DNS label boundaries, IP literals, CIDRs,
// optional ports, and the whole-entry wildcard; a leading *. aliases its hostname.
type Proxy struct {
	URI      url.URL
	Username string
	Password string
	NoProxy  []string
}

// API contains resolved API endpoints.
type API struct {
	Heartbeat Endpoint
}

// Endpoint contains a resolved endpoint URL and TLS settings.
type Endpoint struct {
	URI       url.URL
	TLSVerify bool
	CAPath    string
}

// resolve converts merged native settings into complete resolved sections.
func (partial dtoConfig) resolve() (Config, error) {
	if partial.HTTP.Timeout.Connect == nil || partial.HTTP.Timeout.Request == nil || partial.HTTP.Timeout.Idle == nil ||
		partial.HTTP.Proxy.URI == nil || partial.HTTP.Proxy.Username == nil || partial.HTTP.Proxy.Password == nil ||
		partial.HTTP.Proxy.NoProxy == nil || partial.API.Heartbeat.URI == nil ||
		partial.API.Heartbeat.TLSVerify == nil || partial.API.Heartbeat.CAPath == nil {
		return Config{}, errors.New("native configuration is incomplete")
	}

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

	parsed, err := parseHTTPURL(*partial.URI)
	if err != nil {
		return Endpoint{}, fmt.Errorf("resolve api.heartbeat.uri: %w", err)
	}

	return Endpoint{URI: *parsed, TLSVerify: *partial.TLSVerify, CAPath: *partial.CAPath}, nil
}

// resolveProxy parses the configured proxy URI and resolves optional credentials and bypass rules.
func resolveProxy(partial dtoProxy) (Proxy, error) {
	if partial.URI == nil || partial.Username == nil || partial.Password == nil || partial.NoProxy == nil {
		return Proxy{}, errors.New("resolve http.proxy: required setting is missing")
	}

	var uri url.URL

	if *partial.URI != "" {
		parsed, err := parseHTTPURL(*partial.URI)
		if err != nil {
			return Proxy{}, fmt.Errorf("resolve http.proxy.uri: %w", err)
		}

		uri = *parsed
	} else if *partial.Username != "" || *partial.Password != "" {
		return Proxy{}, errors.New("resolve http.proxy: credentials require a proxy URI")
	}

	return Proxy{
		URI:      uri,
		Username: *partial.Username,
		Password: *partial.Password,
		NoProxy:  append([]string{}, (*partial.NoProxy)...),
	}, nil
}
