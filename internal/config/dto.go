package config

import (
	"errors"
	"log/slog"
	"math"
	"net"
	"net/url"
	"strings"
	"time"
)

var errInvalidURL = errors.New("invalid HTTP(S) URL")

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
	URI      *string   `toml:"uri"`
	Username *string   `toml:"username"`
	Password *string   `toml:"password"`
	NoProxy  *[]string `toml:"no-proxy"`
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

// update recursively validates and merges supplied native configuration leaves.
func (partial *dtoConfig) update(next dtoConfig, source string) {
	partial.HTTP.update(next.HTTP, source)
	partial.API.update(next.API, source)
}

// update validates and merges timeout and proxy configuration sections.
func (partial *dtoHTTP) update(next dtoHTTP, source string) {
	partial.Timeout.update(next.Timeout, source)
	partial.Proxy.update(next.Proxy, source)
}

// update validates and merges nonnegative timeout values in seconds.
func (partial *dtoTimeout) update(next dtoTimeout, source string) {
	if next.Connect != nil {
		if validSeconds(*next.Connect) {
			partial.Connect = new(*next.Connect)
		} else {
			reject(source, "http.timeout.connect")
		}
	}

	if next.Request != nil {
		if validSeconds(*next.Request) {
			partial.Request = new(*next.Request)
		} else {
			reject(source, "http.timeout.request")
		}
	}

	if next.Idle != nil {
		if validSeconds(*next.Idle) {
			partial.Idle = new(*next.Idle)
		} else {
			reject(source, "http.timeout.idle")
		}
	}
}

// update validates and merges proxy URLs, credentials, and bypass rules.
func (partial *dtoProxy) update(next dtoProxy, source string) {
	if next.URI != nil {
		if value := checkedURI(*next.URI, source, "http.proxy.uri"); value != nil {
			partial.URI = value
		}
	}

	if next.Username != nil {
		partial.Username = new(strings.TrimSpace(*next.Username))
	}

	if next.Password != nil {
		partial.Password = new(*next.Password)
	}

	if next.NoProxy != nil {
		partial.NoProxy = new(normalizeNoProxy(*next.NoProxy, source))
	}
}

// update validates and merges partial API endpoint settings.
func (partial *dtoAPI) update(next dtoAPI, source string) {
	partial.Heartbeat.update(next.Heartbeat, source)
}

// update validates and merges endpoint URL and TLS settings.
func (partial *dtoEndpoint) update(next dtoEndpoint, source string) {
	if next.URI != nil {
		if value := checkedURI(*next.URI, source, "api.heartbeat.uri"); value != nil {
			partial.URI = value
		}
	}

	if next.TLSVerify != nil {
		partial.TLSVerify = new(*next.TLSVerify)
	}

	if next.CAPath != nil {
		partial.CAPath = new(strings.TrimSpace(*next.CAPath))
	}
}

// checkedURI validates a supplied URL, preserving explicit emptiness for final resolution.
func checkedURI(value, source, field string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return new("")
	}

	parsed, err := parseHTTPURL(value)
	if err != nil {
		reject(source, field)

		return nil
	}

	if strings.EqualFold(parsed.Scheme, "http") {
		slog.Warn("configuration uses an unencrypted HTTP URI", "source", source, "field", field)
	}

	return new(value)
}

// parseHTTPURL validates an absolute HTTP(S) URL without embedded credentials or request data.
func parseHTTPURL(value string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil ||
		(!strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https")) ||
		parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errInvalidURL
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)

	return parsed, nil
}

// validSeconds reports whether seconds can be represented as a time.Duration.
func validSeconds(seconds int64) bool {
	return seconds >= 0 && seconds <= math.MaxInt64/int64(time.Second)
}

// reject logs an invalid setting without disclosing its configured value.
func reject(source, field string) {
	slog.Debug("ignoring invalid configuration value", "source", source, "field", field)
}

// validateBypassRule reports whether rule follows the documented no-proxy syntax.
func validateBypassRule(rule string) bool {
	if rule == "*" {
		return true
	}

	rule = strings.TrimPrefix(rule, "*.")
	if rule == "" || strings.Contains(rule, "*") {
		return false
	}

	if strings.Contains(rule, "/") {
		return validateCIDRRule(rule)
	}

	host := rule
	switch {
	case strings.HasPrefix(rule, "["):
		parsedHost, port, err := net.SplitHostPort(rule)
		switch {
		case err == nil:
			if !validRulePort(port) {
				return false
			}

			host = parsedHost
		case strings.HasSuffix(rule, "]"):
			host = strings.Trim(rule, "[]")
		default:
			return false
		}
	case strings.Count(rule, ":") == 1:
		parts := strings.SplitN(rule, ":", 2)
		if !validRulePort(parts[1]) {
			return false
		}

		host = parts[0]
	case strings.Count(rule, ":") > 1:
		return net.ParseIP(rule) != nil
	}

	if host == "" {
		return false
	}

	if _, _, err := net.ParseCIDR(host); err == nil {
		return true
	}

	if net.ParseIP(host) != nil {
		return true
	}

	return validDNSName(strings.TrimSuffix(host, "."))
}

// validateCIDRRule checks an IP range, optionally followed by a port.
func validateCIDRRule(rule string) bool {
	if _, _, err := net.ParseCIDR(rule); err == nil {
		return true
	}

	colon := strings.LastIndex(rule, ":")
	if colon <= strings.LastIndex(rule, "/") || !validRulePort(rule[colon+1:]) {
		return false
	}

	_, _, err := net.ParseCIDR(rule[:colon])

	return err == nil
}

// validRulePort reports whether value is a decimal TCP port.
func validRulePort(value string) bool {
	if value == "" {
		return false
	}

	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}

	port := 0
	for _, char := range value {
		port = port*10 + int(char-'0')
		if port > 65535 {
			return false
		}
	}

	return port > 0
}

// validDNSName checks the DNS label syntax used by bypass host rules.
func validDNSName(host string) bool {
	if host == "" {
		return false
	}

	for label := range strings.SplitSeq(host, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}

		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') &&
				(char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}

	return true
}

// normalizeNoProxy trims, validates, and copies bypass rules while logging rejected entries.
func normalizeNoProxy(rules []string, source string) []string {
	result := make([]string, 0, len(rules))
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		if !validateBypassRule(rule) {
			slog.Debug("ignoring invalid no-proxy rule", "source", source, "field", "http.proxy.no-proxy")

			continue
		}

		result = append(result, rule)
	}

	return result
}
