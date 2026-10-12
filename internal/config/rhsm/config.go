package rhsm

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// TODO Should we return parsed url.URL objects since it will have to be done anyway?

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
	NoProxy       string
}

// RHSM contains resolved CA settings.
type RHSM struct {
	CACertDir  string
	RepoCACert string
}

// resolve converts merged RHSM values into complete resolved sections.
func (partial dtoConfig) resolve() (Config, error) {
	server := partial.Server

	candlepin, err := resolveURI("https", *server.Hostname, *server.Port, *server.Prefix)
	if err != nil {
		return Config{}, fmt.Errorf("resolve server: %w", err)
	}

	candlepinPort, err := strconv.ParseUint(*server.Port, 10, 16)
	if err != nil {
		return Config{}, fmt.Errorf("parse server port: %w", err)
	}

	proxy, err := resolveURI(*server.ProxyScheme, *server.ProxyHostname, *server.ProxyPort, "")
	if err != nil {
		return Config{}, fmt.Errorf("resolve proxy: %w", err)
	}

	proxyPassword, _ := proxy.User.Password()

	caDir := *partial.RHSM.CACertDir
	repoCACert := strings.ReplaceAll(*partial.RHSM.RepoCACert, "%(ca_cert_dir)s", caDir)

	return Config{
		Server: Server{
			Hostname:      candlepin.Hostname(),
			Prefix:        candlepin.Path,
			Port:          uint16(candlepinPort),
			TLSVerify:     !*server.Insecure,
			ProxyHostname: proxy.Host,
			ProxyScheme:   proxy.Scheme,
			ProxyUser:     proxy.User.Username(),
			ProxyPassword: proxyPassword,
			NoProxy:       *server.NoProxy,
		},
		RHSM: RHSM{
			CACertDir:  caDir,
			RepoCACert: repoCACert,
		},
	}, nil
}

func resolveURI(scheme, hostname, port, prefix string) (url.URL, error) {
	if hostname == "" {
		return url.URL{}, nil
	}

	if scheme == "" {
		scheme = "https"
	}

	parsed := &url.URL{Scheme: scheme, Host: net.JoinHostPort(hostname, port), Path: prefix}

	return *parsed, nil
}
