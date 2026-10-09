package rhsm

import (
	"errors"
	"fmt"
	"os"
	"strings"

	internalfs "github.com/m-horky/rhc-heartbeat/internal/fs"
	"gopkg.in/ini.v1"
)

// Load merges embedded defaults with the RHSM file and resolves the result.
func Load(filesystem internalfs.FS, path string) (Config, error) {
	defaultDTO, err := defaults()
	if err != nil {
		return Config{}, fmt.Errorf("decode embedded RHSM defaults: %w", err)
	}

	var partial dtoConfig
	partial.update(defaultDTO, "embedded defaults")

	if _, err := partial.resolve(); err != nil {
		return Config{}, fmt.Errorf("invalid embedded RHSM defaults: %w", err)
	}

	raw, err := filesystem.Read(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("read RHSM configuration %s: %w", path, err)
		}
	} else {
		parsed, parseErr := parse(raw)
		if parseErr != nil {
			return Config{}, fmt.Errorf("parse RHSM configuration %s: %w", path, parseErr)
		}

		partial.update(parsed, path)
	}

	resolved, err := partial.resolve()
	if err != nil {
		return Config{}, fmt.Errorf("resolve RHSM configuration %s: %w", path, err)
	}

	return resolved, nil
}

// parse decodes an INI document into optional RHSM configuration leaves.
func parse(data []byte) (dtoConfig, error) {
	file, err := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true}, data)
	if err != nil {
		return dtoConfig{}, errors.New("invalid INI syntax")
	}

	var result dtoConfig

	for _, sectionName := range []string{"server", "rhsm"} {
		section, sectionErr := file.GetSection(sectionName)
		if sectionErr != nil {
			continue
		}

		for _, key := range section.Keys() {
			value := key.Value()

			name := strings.ToLower(key.Name())
			if sectionName == "server" {
				setServerValue(&result.Server, name, value)
			} else {
				setRHSMValue(&result.RHSM, name, value)
			}
		}
	}

	return result, nil
}

// setServerValue records known server keys while ignoring unknown INI keys.
func setServerValue(server *dtoServer, name, value string) {
	switch name {
	case "hostname":
		server.Hostname = new(value)
	case "prefix":
		server.Prefix = new(value)
	case "port":
		server.Port = new(value)
	case "insecure":
		server.Insecure = new(value)
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
}

// setRHSMValue records known CA keys while ignoring unknown INI keys.
func setRHSMValue(section *dtoRHSM, name, value string) {
	switch name {
	case "ca_cert_dir":
		section.CACertDir = new(value)
	case "repo_ca_cert":
		section.RepoCACert = new(value)
	}
}
