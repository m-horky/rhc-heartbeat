package config

import (
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/m-horky/rhc-heartbeat/internal/config/rhsm"
)

// newPartialFromRHSM translates resolved RHSM settings into a partial native layer.
func newPartialFromRHSM(source rhsm.Config) (dtoConfig, error) {
	endpoint, err := buildTelemetryURI(source.Server)
	if err != nil {
		return dtoConfig{}, err
	}

	proxy, err := buildProxyURI(source.Server)
	if err != nil {
		return dtoConfig{}, err
	}

	return dtoConfig{
		HTTP: dtoHTTP{Proxy: dtoProxy{
			URI:     new(proxy.String()),
			NoProxy: new(source.Server.NoProxy),
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
	labels := strings.Split(server.Hostname, ".")
	if hasLabelSuffix(labels, []string{"rhsm", "stage", "redhat", "com"}) {
		return url.URL{
			Scheme: "https",
			Host:   buildURIHost("cert.console.stage.redhat.com", server.Port),
			Path:   "/api/rhel-telemetry/v1/receive",
		}, nil
	}

	if hasLabelSuffix(labels, []string{"rhsm", "redhat", "com"}) {
		return url.URL{
			Scheme: "https",
			Host:   buildURIHost("cert.console.redhat.com", server.Port),
			Path:   "/api/rhel-telemetry/v1/receive",
		}, nil
	}

	return url.URL{
		Scheme: "https",
		Host:   buildURIHost(server.Hostname, server.Port),
		Path:   "/prometheus/api/v1/write",
	}, nil
}

// buildURIHost omits :443 in a 'host:port' string.
func buildURIHost(hostname string, port uint16) string {
	if port == 443 {
		return hostname
	}

	return net.JoinHostPort(hostname, strconv.Itoa(int(port)))
}

// buildProxyURI constructs the RHSM-derived proxy URL, or a zero URL when disabled.
//
// If the port is missing, it defaults to 3128.
func buildProxyURI(server rhsm.Server) (url.URL, error) {
	host := strings.TrimSpace(server.ProxyHostname)
	if host == "" {
		return url.URL{}, nil
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

	proxy := url.URL{
		Scheme: scheme,
		Host:   net.JoinHostPort(host, strconv.Itoa(int(port))),
	}
	if server.ProxyUser != "" || server.ProxyPassword != "" {
		proxy.User = url.UserPassword(server.ProxyUser, server.ProxyPassword)
	}

	return proxy, nil
}

// hasLabelSuffix compares DNS labels against a provided DNS label suffix.
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
