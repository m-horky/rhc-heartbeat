package config

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/m-horky/rhc-heartbeat/internal/config/rhsm"
)

// fromRHSM translates resolved RHSM settings into a partial native layer.
func fromRHSM(source rhsm.Config) (dtoConfig, error) {
	endpoint, err := buildTelemetryURI(source.Server)
	if err != nil {
		return dtoConfig{}, err
	}

	proxy, err := buildProxyURI(source.Server)
	if err != nil {
		return dtoConfig{}, err
	}

	noProxy := append([]string{}, source.Server.NoProxy...)

	proxyUser, proxyPassword := "", ""
	if proxy.String() != "" {
		proxyUser = source.Server.ProxyUser
		proxyPassword = source.Server.ProxyPassword
	}

	return dtoConfig{
		HTTP: dtoHTTP{Proxy: dtoProxy{
			URI:      new(proxy.String()),
			Username: new(proxyUser),
			Password: new(proxyPassword),
			NoProxy:  new(noProxy),
		}},
		API: dtoAPI{Heartbeat: dtoEndpoint{
			URI:       new(endpoint.String()),
			TLSVerify: new(source.Server.TLSVerify),
			CAPath:    new(source.RHSM.RepoCACert),
		}},
	}, nil
}

// buildTelemetryURI derives the telemetry endpoint from the RHSM server hostname.
func buildTelemetryURI(server rhsm.Server) (url.URL, error) {
	host := strings.TrimSpace(server.Hostname)
	if !isValidHostname(host) {
		return url.URL{}, errors.New("invalid RHSM server hostname")
	}

	normalized := strings.TrimSuffix(host, ".")

	labels := strings.Split(normalized, ".")
	if hasLabelSuffix(labels, []string{"rhsm", "stage", "redhat", "com"}) {
		return url.URL{Scheme: "https", Host: "cert.console.stage.redhat.com", Path: "/api/rhel-telemetry/v1/receive"}, nil
	}

	if hasLabelSuffix(labels, []string{"rhsm", "redhat", "com"}) {
		return url.URL{Scheme: "https", Host: "cert.console.redhat.com", Path: "/api/rhel-telemetry/v1/receive"}, nil
	}

	if server.Port == 0 {
		return url.URL{}, errors.New("invalid RHSM server port")
	}

	return url.URL{
		Scheme: "https",
		Host:   net.JoinHostPort(host, strconv.Itoa(int(server.Port))),
		Path:   "/prometheus/api/v1/write",
	}, nil
}

// buildProxyURI constructs the RHSM-derived proxy URL, or a zero URL when disabled.
func buildProxyURI(server rhsm.Server) (url.URL, error) {
	host := strings.TrimSpace(server.ProxyHostname)
	if host == "" {
		return url.URL{}, nil
	}

	if !isValidHostname(host) {
		return url.URL{}, errors.New("invalid RHSM proxy hostname")
	}

	port := server.ProxyPort
	if port == 0 {
		port = 3128
	}

	scheme := strings.ToLower(strings.TrimSpace(server.ProxyScheme))
	if scheme == "" {
		scheme = "http"
	}

	if scheme != "http" && scheme != "https" {
		return url.URL{}, errors.New("invalid RHSM proxy scheme")
	}

	return url.URL{Scheme: scheme, Host: net.JoinHostPort(host, strconv.Itoa(int(port)))}, nil
}

// isValidHostname validates the hostname used to construct a network URL.
func isValidHostname(host string) bool {
	if host == "" || strings.ContainsAny(host, "/?# \t\r\n") {
		return false
	}

	if net.ParseIP(host) != nil {
		return true
	}

	trimmed := strings.TrimSuffix(host, ".")
	if trimmed == "" {
		return false
	}

	for label := range strings.SplitSeq(trimmed, ".") {
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

// hasLabelSuffix compares normalized DNS labels instead of arbitrary string suffixes.
func hasLabelSuffix(host, suffix []string) bool {
	if len(host) < len(suffix) {
		return false
	}

	offset := len(host) - len(suffix)
	for index, label := range suffix {
		if !strings.EqualFold(host[offset+index], label) {
			return false
		}
	}

	return true
}
