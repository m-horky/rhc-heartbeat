package rhsm

import (
	"errors"
	"fmt"
	"os"
	"strings"

	internalfs "github.com/m-horky/rhc-heartbeat/internal/fs"
	"gopkg.in/ini.v1"
)

// Load merges the rhsm.conf defaults with /etc/rhsm/rhsm.conf.
func Load(filesystem internalfs.FS, path string) (Config, error) {
	partial, err := newPartialFromDefaults()
	if err != nil {
		return Config{}, fmt.Errorf("read default rhsm.conf: %w", err)
	}

	// merge rhsm.conf
	raw, err := filesystem.Read(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("read rhsm.conf %s: %w", path, err)
		}
	} else {
		parsed, parseErr := newPartialFromINI(raw)
		if parseErr != nil {
			return Config{}, fmt.Errorf("parse rhsm.conf %s: %w", path, parseErr)
		}

		if err := partial.update(parsed); err != nil {
			return Config{}, fmt.Errorf("merge rhsm.conf %s: %w", path, err)
		}
	}

	// resolve and validate
	resolved, err := partial.resolve()
	if err != nil {
		return Config{}, fmt.Errorf("resolve rhsm.conf %s: %w", path, err)
	}

	return resolved, nil
}

// newPartialFromINI decodes an INI document into configuration partial.
func newPartialFromINI(data []byte) (*dtoConfig, error) {
	file, err := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true}, data)
	if err != nil {
		return nil, errors.New("invalid INI syntax")
	}

	result := &dtoConfig{
		Server: dtoServer{},
		RHSM:   dtoRHSM{},
	}

	for _, sectionName := range []string{"server", "rhsm"} {
		section, sectionErr := file.GetSection(sectionName)
		if sectionErr != nil {
			continue
		}

		for _, key := range section.Keys() {
			value := key.Value()

			name := strings.ToLower(key.Name())
			if sectionName == "server" {
				err := setServerValue(&result.Server, name, value)
				if err != nil {
					return nil, fmt.Errorf("interpret value: %w", err)
				}
			} else {
				err := setRHSMValue(&result.RHSM, name, value)
				if err != nil {
					return nil, fmt.Errorf("interpret value: %w", err)
				}
			}
		}
	}

	return result, nil
}

// setServerValue records known server keys.
func setServerValue(server *dtoServer, name, value string) error {
	switch name {
	case "hostname":
		server.Hostname = new(value)
	case "prefix":
		server.Prefix = new(value)
	case "port":
		server.Port = new(value)
	case "insecure":
		parsed, err := parseBool(value)
		if err != nil {
			return err
		}

		server.Insecure = new(parsed)
	case "proxy_hostname":
		server.ProxyHostname = new(value)
	case "proxy_scheme":
		server.ProxyScheme = new(value)
	case "proxy_port":
		server.ProxyPort = new(value)
	case "proxy_user":
		server.ProxyUser = new(value)
	case "proxy_password":
		server.ProxyPassword = new(value)
	case "no_proxy":
		server.NoProxy = new(value)
	}

	return nil
}

// setRHSMValue records known certificate authority keys.
func setRHSMValue(section *dtoRHSM, name, value string) error {
	switch name {
	case "ca_cert_dir":
		section.CACertDir = new(value)
	case "repo_ca_cert":
		section.RepoCACert = new(value)
	}

	return nil
}

func parseBool(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "true", "1":
		return true, nil
	case "false", "0":
		return false, nil
	default:
		return false, fmt.Errorf("invalid value: %s", value)
	}
}
