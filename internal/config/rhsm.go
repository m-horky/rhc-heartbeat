package config

import (
	"errors"
	"fmt"
	iofs "io/fs"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"gopkg.in/ini.v1"
)

// rhsmConfiguration is the private DTO for the relevant portions of rhsm.conf.
type rhsmConfiguration struct {
	Server rhsmServer `ini:"server"`
	RHSM   rhsmRHSM   `ini:"rhsm"`
	Proxy  rhsmProxy  `ini:"proxy"`
}

type rhsmServer struct {
	Hostname string `ini:"hostname"`
	Port     string `ini:"port"`
	Insecure string `ini:"insecure"`
}

type rhsmRHSM struct {
	BaseURL    string `ini:"baseurl"`
	RepoCACert string `ini:"repo_ca_cert"`
}

type rhsmProxy struct {
	Hostname string `ini:"proxy_hostname"`
	Port     string `ini:"proxy_port"`
	User     string `ini:"proxy_user"`
	Password string `ini:"proxy_password"`
}

type rhsmSettings struct {
	CandlepinURI    string
	RemoteWriteURI  string
	Insecure        bool
	InsecurePresent bool
	CAPath          string
	Proxy           Proxy
}

// loadRHSM reads, parses, and translates the supported rhsm.conf settings.
func loadRHSM(filesystem fs.FS, path string) (rhsmSettings, error) {
	data, err := filesystem.Read(path)
	if err != nil {
		if errors.Is(err, iofs.ErrNotExist) {
			return rhsmSettings{}, nil
		}

		return rhsmSettings{}, fmt.Errorf("read legacy configuration %s: %w", path, err)
	}

	file, err := ini.Load(data)
	if err != nil {
		return rhsmSettings{}, fmt.Errorf("parse legacy configuration %s: invalid INI syntax", path)
	}

	var legacy rhsmConfiguration
	if err := file.MapTo(&legacy); err != nil {
		return rhsmSettings{}, fmt.Errorf("map legacy configuration %s: %w", path, err)
	}

	settings, err := mapRHSM(legacy)
	if err != nil {
		return rhsmSettings{}, fmt.Errorf("legacy configuration %s: %w", path, err)
	}

	return settings, nil
}

// mapRHSM translates parsed legacy sections into heartbeat endpoint settings.
func mapRHSM(legacy rhsmConfiguration) (rhsmSettings, error) {
	insecure, insecurePresent, err := parseRHSMBoolean(legacy.Server.Insecure)
	if err != nil {
		return rhsmSettings{}, rhsmError("server.insecure", err)
	}

	candlepinURI, remoteWriteURI, err := mapRHSMServer(legacy.Server, insecurePresent)
	if err != nil {
		return rhsmSettings{}, err
	}

	proxy, err := mapRHSMProxy(legacy.Proxy)
	if err != nil {
		return rhsmSettings{}, err
	}

	return rhsmSettings{
		CandlepinURI:    candlepinURI,
		RemoteWriteURI:  remoteWriteURI,
		Insecure:        insecure,
		InsecurePresent: insecurePresent,
		CAPath:          strings.TrimSpace(legacy.RHSM.RepoCACert),
		Proxy:           proxy,
	}, nil
}

// mapRHSMServer builds Candlepin and Remote Write URIs from the legacy server section.
func mapRHSMServer(server rhsmServer, insecurePresent bool) (string, string, error) {
	host := strings.TrimSpace(server.Hostname)
	if host == "" {
		if strings.TrimSpace(server.Port) != "" || insecurePresent {
			return "", "", rhsmError("server.hostname", errors.New("missing hostname"))
		}

		return "", "", nil
	}

	if _, err := parseRHSMHost(host); err != nil {
		return "", "", rhsmError("server.hostname", err)
	}

	port, err := parseRHSMPort(server.Port)
	if err != nil {
		return "", "", rhsmError("server.port", err)
	}

	candlepin := "https://" + net.JoinHostPort(host, strconv.Itoa(port))
	if _, err := parseRHSMURL(candlepin); err != nil {
		return "", "", rhsmError("server", err)
	}

	return candlepin, appendRemoteWritePath(candlepin), nil
}

// appendRemoteWritePath appends the Remote Write route to a validated Candlepin URI.
func appendRemoteWritePath(candlepinURI string) string {
	return strings.TrimRight(candlepinURI, "/") + "/api/v1/write"
}

// mapRHSMProxy converts legacy proxy fields into HTTP transport settings.
func mapRHSMProxy(proxy rhsmProxy) (Proxy, error) {
	host := strings.TrimSpace(proxy.Hostname)
	portValue := strings.TrimSpace(proxy.Port)
	result := Proxy{}

	if host != "" || portValue != "" {
		if host == "" {
			return Proxy{}, rhsmError("proxy.proxy_hostname", errors.New("missing hostname"))
		}

		if _, err := parseRHSMHost(host); err != nil {
			return Proxy{}, rhsmError("proxy.proxy_hostname", err)
		}

		port, err := parseRHSMPort(portValue)
		if err != nil {
			return Proxy{}, rhsmError("proxy.proxy_port", err)
		}

		result.URI = "https://" + net.JoinHostPort(host, strconv.Itoa(port))
	}

	if value := strings.TrimSpace(proxy.User); value != "" {
		result.User = value
	}

	if proxy.Password != "" {
		result.Password = proxy.Password
	}

	if result.URI != "" {
		if _, err := validateProxyURI(result.URI); err != nil {
			return Proxy{}, rhsmError("proxy", err)
		}
	}

	return result, nil
}

// rhsmError identifies the legacy setting associated with a validation failure.
func rhsmError(key string, err error) error {
	return fmt.Errorf("error reading legacy %s: %w", key, err)
}

// parseRHSMPort parses a legacy port within the valid TCP port range.
func parseRHSMPort(value string) (int, error) {
	value = strings.TrimSpace(value)

	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65535 {
		return 0, errors.New("invalid port")
	}

	return port, nil
}

// parseRHSMBoolean parses supported legacy boolean spellings and reports presence.
func parseRHSMBoolean(value string) (bool, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return false, false, nil
	}

	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true, true, nil
	case "0", "false", "no", "off":
		return false, true, nil
	default:
		return false, true, errors.New("invalid boolean")
	}
}

// parseRHSMHost rejects values that cannot safely be used as URL hosts.
func parseRHSMHost(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, " /?#") {
		return "", errors.New("invalid host")
	}

	if ip := net.ParseIP(value); ip == nil && strings.Contains(value, ":") {
		return "", errors.New("invalid host")
	}

	return value, nil
}

// parseRHSMURL accepts HTTPS URLs without credentials, queries, or fragments.
func parseRHSMURL(value string) (string, error) {
	value = strings.TrimSpace(value)

	u, err := url.ParseRequestURI(value)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil ||
		strings.Contains(value, "#") || u.Fragment != "" || u.RawQuery != "" {
		return "", errors.New("invalid HTTPS URL")
	}

	return value, nil
}
