package rhsm

import (
	"errors"
	"fmt"
	"strings"
)

// Config contains resolved RHSM configuration.
type Config struct {
	Server Server
	RHSM   RHSM
}

// Server contains resolved subscription-server and proxy settings.
type Server struct {
	Hostname      string
	Prefix        string
	Port          uint16
	TLSVerify     bool
	ProxyHostname string
	ProxyScheme   string
	ProxyPort     uint16
	ProxyUser     string
	ProxyPassword string
	NoProxy       []string
}

// RHSM contains resolved CA settings.
type RHSM struct {
	CACertDir  string
	RepoCACert string
}

// resolve converts merged RHSM values into complete resolved sections.
func (partial dtoConfig) resolve() (Config, error) {
	server := partial.Server
	if server.Hostname == nil || server.Prefix == nil || server.Port == nil || server.Insecure == nil ||
		server.ProxyHostname == nil || server.ProxyScheme == nil || server.ProxyPort == nil ||
		server.ProxyUser == nil || server.ProxyPassword == nil || server.NoProxy == nil ||
		partial.RHSM.CACertDir == nil || partial.RHSM.RepoCACert == nil {
		return Config{}, errors.New("RHSM configuration is incomplete")
	}

	port, err := parseUint16(*server.Port)
	if err != nil {
		return Config{}, fmt.Errorf("resolve server.port: %w", err)
	}

	resolved := Server{
		Hostname:      *server.Hostname,
		Prefix:        *server.Prefix,
		Port:          port,
		TLSVerify:     *server.Insecure == "0",
		ProxyHostname: *server.ProxyHostname,
		ProxyScheme:   *server.ProxyScheme,
		ProxyUser:     *server.ProxyUser,
		ProxyPassword: *server.ProxyPassword,
		NoProxy:       splitNoProxy(*server.NoProxy),
	}
	if err := resolveProxyDefaults(&resolved, server); err != nil {
		return Config{}, err
	}

	caDir := *partial.RHSM.CACertDir
	repoCACert := strings.ReplaceAll(*partial.RHSM.RepoCACert, "%(ca_cert_dir)s", caDir)

	return Config{
		Server: resolved,
		RHSM: RHSM{
			CACertDir:  caDir,
			RepoCACert: repoCACert,
		},
	}, nil
}

// resolveProxyDefaults applies RHSM defaults for an enabled proxy.
func resolveProxyDefaults(resolved *Server, partial dtoServer) error {
	if resolved.ProxyHostname == "" {
		return nil
	}

	if resolved.ProxyScheme == "" {
		resolved.ProxyScheme = "http"
	}

	if *partial.ProxyPort == "" {
		resolved.ProxyPort = 3128

		return nil
	}

	port, err := parseUint16(*partial.ProxyPort)
	if err != nil {
		return fmt.Errorf("resolve server.proxy_port: %w", err)
	}

	resolved.ProxyPort = port

	return nil
}

// splitNoProxy converts RHSM's comma-separated bypass list to individual rules.
func splitNoProxy(value string) []string {
	if value == "" {
		return []string{}
	}

	entries := strings.Split(value, ",")

	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry = strings.TrimSpace(entry); entry != "" {
			result = append(result, entry)
		}
	}

	return result
}
