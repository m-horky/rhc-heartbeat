package rhsm

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
)

// dtoConfig contains one partial RHSM configuration layer.
type dtoConfig struct {
	Server dtoServer
	RHSM   dtoRHSM
}

// dtoServer contains optional subscription-server and proxy settings.
type dtoServer struct {
	Hostname      *string
	Prefix        *string // API path prefix
	Port          *string
	Insecure      *string
	ProxyHostname *string
	ProxyScheme   *string
	ProxyPort     *string
	ProxyUser     *string
	ProxyPassword *string
	NoProxy       *string
}

// dtoRHSM contains optional CA settings before INI interpolation.
type dtoRHSM struct {
	CACertDir  *string
	RepoCACert *string
}

// update validates and merges supplied RHSM server leaves into the receiver.
func (partial *dtoConfig) update(next dtoConfig, source string) {
	partial.Server.update(next.Server, source)
	partial.RHSM.update(next.RHSM, source)
}

// update validates and merges subscription-server and proxy leaves.
func (partial *dtoServer) update(next dtoServer, source string) {
	partial.updateURI(next, source)

	if next.Port != nil {
		if isValidPort(*next.Port, false) {
			partial.Port = new(strings.TrimSpace(*next.Port))
		} else {
			reject(source, "server.port")
		}
	}

	if next.Insecure != nil {
		value := strings.ToLower(strings.TrimSpace(*next.Insecure))
		switch value {
		case "0", "false":
			partial.Insecure = new("0")
		case "1", "true":
			partial.Insecure = new("1")
		default:
			reject(source, "server.insecure")
		}
	}

	if next.ProxyHostname != nil {
		value := strings.TrimSpace(*next.ProxyHostname)
		if value != "" && !isValidHostname(value) {
			reject(source, "server.proxy_hostname")
		} else {
			partial.ProxyHostname = new(value)
		}
	}

	if next.ProxyScheme != nil {
		value := strings.ToLower(strings.TrimSpace(*next.ProxyScheme))
		if value != "" && value != "http" && value != "https" {
			reject(source, "server.proxy_scheme")
		} else {
			partial.ProxyScheme = new(value)
		}
	}

	if next.ProxyPort != nil {
		value := strings.TrimSpace(*next.ProxyPort)
		if isValidPort(value, true) {
			partial.ProxyPort = new(value)
		} else {
			reject(source, "server.proxy_port")
		}
	}

	partial.updateProxyCredentials(next, source)
}

// updateURI validates and merges the RHSM server hostname and prefix.
func (partial *dtoServer) updateURI(next dtoServer, source string) {
	if next.Hostname != nil {
		value := strings.TrimSpace(*next.Hostname)
		if value == "" || !isValidHostname(value) {
			reject(source, "server.hostname")
		} else {
			partial.Hostname = new(value)
		}
	}

	if next.Prefix != nil {
		partial.Prefix = new(strings.TrimSpace(*next.Prefix))
	}
}

// updateProxyCredentials merges RHSM proxy authentication and bypass settings.
func (partial *dtoServer) updateProxyCredentials(next dtoServer, _ string) {
	if next.ProxyUser != nil {
		partial.ProxyUser = new(strings.TrimSpace(*next.ProxyUser))
	}

	if next.ProxyPassword != nil {
		partial.ProxyPassword = new(*next.ProxyPassword)
	}

	if next.NoProxy != nil {
		partial.NoProxy = new(*next.NoProxy)
	}
}

// update merges RHSM CA settings without interpreting interpolation syntax.
func (partial *dtoRHSM) update(next dtoRHSM, _ string) {
	if next.CACertDir != nil {
		partial.CACertDir = new(*next.CACertDir)
	}

	if next.RepoCACert != nil {
		partial.RepoCACert = new(*next.RepoCACert)
	}
}

// reject logs a semantic setting error without disclosing its value.
func reject(source, field string) {
	slog.Debug("ignoring invalid configuration value", "source", source, "field", field)
}

// isValidPort reports whether value is empty when allowed or a valid TCP port.
func isValidPort(value string, emptyAllowed bool) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return emptyAllowed
	}

	port, err := strconv.Atoi(value)

	return err == nil && port >= 1 && port <= 65535
}

// isValidHostname checks that value is a DNS hostname or an IP literal.
func isValidHostname(value string) bool {
	value = strings.TrimSuffix(value, ".")
	if value == "" || strings.ContainsAny(value, "/?#[] \t\r\n") {
		return false
	}

	if net.ParseIP(value) != nil {
		return true
	}

	for label := range strings.SplitSeq(value, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}

		for _, character := range label {
			if (character < 'a' || character > 'z') && (character < 'A' || character > 'Z') &&
				(character < '0' || character > '9') && character != '-' {
				return false
			}
		}
	}

	return true
}

// parseUint16 parses a previously validated TCP port.
func parseUint16(value string) (uint16, error) {
	port, err := strconv.ParseUint(value, 10, 16)
	if err != nil || port == 0 {
		return 0, fmt.Errorf("invalid port")
	}

	return uint16(port), nil
}
