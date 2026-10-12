package rhsm

import (
	"errors"
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

	if server.Hostname == nil || server.Port == nil {
		return Config{}, errors.New("resolve server: hostname and port are required")
	}

	candlepin := url.URL{
		Scheme: "https",
		Host:   net.JoinHostPort(*server.Hostname, *server.Port),
		Path:   *server.Prefix,
	}

	candlepinPort, err := strconv.ParseUint(*server.Port, 10, 16)
	if err != nil {
		return Config{}, fmt.Errorf("parse server port: %w", err)
	}

	var proxyPort uint64
	if server.ProxyHostname != nil && *server.ProxyHostname != "" && server.ProxyPort != nil && *server.ProxyPort != "" {
		proxyPort, err = strconv.ParseUint(*server.ProxyPort, 10, 16)
		if err != nil {
			return Config{}, fmt.Errorf("parse proxy port: %w", err)
		}
	}

	caDir := *partial.RHSM.CACertDir
	repoCACert := strings.ReplaceAll(*partial.RHSM.RepoCACert, "%(ca_cert_dir)s", caDir)

	return Config{
		Server: Server{
			Hostname:      candlepin.Hostname(),
			Prefix:        candlepin.Path,
			Port:          uint16(candlepinPort),
			TLSVerify:     !*server.Insecure,
			ProxyHostname: *server.ProxyHostname,
			ProxyScheme:   *server.ProxyScheme,
			ProxyPort:     uint16(proxyPort),
			ProxyUser:     *server.ProxyUser,
			ProxyPassword: *server.ProxyPassword,
			NoProxy:       *server.NoProxy,
		},
		RHSM: RHSM{
			CACertDir:  caDir,
			RepoCACert: repoCACert,
		},
	}, nil
}
