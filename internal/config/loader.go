package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/m-horky/rhc-heartbeat/internal/config/rhsm"
	internalfs "github.com/m-horky/rhc-heartbeat/internal/fs"
)

// Load merges defaults, optional RHSM settings, the main file, and drop-ins.
func Load(filesystem internalfs.FS, path, dropInDir string, rhsmFallback *rhsm.Config) (Config, error) {
	partial, err := newPartialFromDefaults()
	if err != nil {
		return Config{}, fmt.Errorf("cannot read default configuration: %w", err)
	}

	// merge rhsm.conf, if present
	if rhsmFallback != nil {
		rhsmPartial, err := newPartialFromRHSM(*rhsmFallback)
		if err != nil {
			return Config{}, fmt.Errorf("cannot convert RHSM configuration: %w", err)
		}

		if err := partial.update(&rhsmPartial); err != nil {
			return Config{}, fmt.Errorf("cannot merge RHSM configuration: %w", err)
		}
	}

	// merge main file
	{
		mainPartial, err := newPartialFromFile(filesystem, path)
		if err != nil {
			return Config{}, fmt.Errorf("cannot parse main configuration: %w", err)
		}

		if err := partial.update(mainPartial); err != nil {
			return Config{}, fmt.Errorf("cannot merge main configuration: %w", err)
		}
	}

	// merge drop-in files
	dropIns, err := discoverDropIns(filesystem, dropInDir)
	if err != nil {
		return Config{}, err
	}

	for _, dropin := range dropIns {
		dropinPartial, err := newPartialFromFile(filesystem, dropin)
		if err != nil {
			return Config{}, fmt.Errorf("drop-in configuration %s: %w", dropin, err)
		}

		if err := partial.update(dropinPartial); err != nil {
			return Config{}, fmt.Errorf("merge drop-in configuration %s: %w", dropin, err)
		}
	}

	// resolve and validate
	resolved, err := partial.resolve()
	if err != nil {
		return Config{}, fmt.Errorf("resolve native configuration: %w", err)
	}

	return resolved, nil
}

// discoverDropIns finds paths of drop-ins.
func discoverDropIns(filesystem internalfs.FS, directory string) ([]string, error) {
	entries, err := filesystem.ReadDir(directory)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return []string{}, nil
	}

	if err != nil {
		return nil, fmt.Errorf("cannot discover drop-in files: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	var result []string

	for _, entry := range entries {
		entryPath := filepath.Join(directory, entry.Name())
		if !strings.HasSuffix(strings.ToLower(entryPath), ".conf") {
			continue
		}

		result = append(result, entryPath)
	}

	return result, nil
}

// newPartialFromFile reads a file and decodes its content into a partial native configuration.
func newPartialFromFile(filesystem internalfs.FS, path string) (*dtoConfig, error) {
	raw, err := filesystem.Read(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &dtoConfig{}, nil
		}

		return nil, fmt.Errorf("read configuration file %s: %w", path, err)
	}

	parsed, err := newPartialFromTOML(raw)
	if err != nil {
		return nil, fmt.Errorf("parse configuration file %s: %w", path, err)
	}

	return parsed, nil
}

// newPartialFromTOML decodes a TOML document into a partial native configuration.
func newPartialFromTOML(data []byte) (*dtoConfig, error) {
	var result *dtoConfig
	if _, err := toml.Decode(string(data), &result); err != nil {
		return nil, errors.New("invalid TOML syntax or value type")
	}

	return result, nil
}
