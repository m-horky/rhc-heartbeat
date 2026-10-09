package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/m-horky/rhc-heartbeat/internal/config/rhsm"
	internalfs "github.com/m-horky/rhc-heartbeat/internal/fs"
)

// Load merges defaults, optional RHSM settings, the main file, and drop-ins.
func Load(filesystem internalfs.FS, path, dropInDir string, fallback *rhsm.Config) (Config, error) {
	defaultDTO, err := defaults()
	if err != nil {
		return Config{}, fmt.Errorf("decode embedded native defaults: %w", err)
	}

	var partial dtoConfig
	partial.update(defaultDTO, "embedded defaults")

	if _, err := partial.resolve(); err != nil {
		return Config{}, fmt.Errorf("invalid embedded native defaults: %w", err)
	}

	if fallback != nil {
		if err := applyFallback(&partial, *fallback); err != nil {
			return Config{}, err
		}
	}

	if err := applyMainFile(filesystem, &partial, path); err != nil {
		return Config{}, err
	}

	if err := applyDropIns(filesystem, &partial, dropInDir); err != nil {
		return Config{}, err
	}

	resolved, err := partial.resolve()
	if err != nil {
		return Config{}, fmt.Errorf("resolve native configuration: %w", err)
	}

	return resolved, nil
}

// applyFallback converts and merges resolved RHSM values over native defaults.
func applyFallback(partial *dtoConfig, fallback rhsm.Config) error {
	converted, err := fromRHSM(fallback)
	if err != nil {
		return fmt.Errorf("convert RHSM configuration: %w", err)
	}

	partial.update(converted, "RHSM fallback")

	return nil
}

// applyMainFile reads and merges the optional main native configuration document.
func applyMainFile(filesystem internalfs.FS, partial *dtoConfig, path string) error {
	data, err := filesystem.Read(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("read native configuration %s: %w", path, err)
	}

	next, err := parse(data)
	if err != nil {
		return fmt.Errorf("parse native configuration %s: %w", path, err)
	}

	partial.update(next, path)

	return nil
}

// applyDropIns discovers, sorts, and merges native configuration drop-ins.
func applyDropIns(filesystem internalfs.FS, partial *dtoConfig, directory string) error {
	entries, err := filesystem.ReadDir(directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("discover configuration drop-ins %s: %w", directory, err)
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".conf") {
			continue
		}

		if err := applyDropIn(filesystem, partial, filepath.Join(directory, name)); err != nil {
			return err
		}
	}

	return nil
}

// applyDropIn reads and merges one regular drop-in file.
func applyDropIn(filesystem internalfs.FS, partial *dtoConfig, path string) error {
	info, err := filesystem.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Debug("configuration drop-in disappeared", "source", path)

			return nil
		}

		return fmt.Errorf("inspect configuration drop-in %s: %w", path, err)
	}

	if info.IsDir || !info.IsRegular {
		return nil
	}

	data, err := filesystem.Read(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Debug("configuration drop-in disappeared", "source", path)

			return nil
		}

		return fmt.Errorf("read configuration drop-in %s: %w", path, err)
	}

	next, err := parse(data)
	if err != nil {
		return fmt.Errorf("parse configuration drop-in %s: %w", path, err)
	}

	partial.update(next, path)

	return nil
}

// parse decodes a TOML document into a partial native configuration.
func parse(data []byte) (dtoConfig, error) {
	var result dtoConfig
	if _, err := toml.Decode(string(data), &result); err != nil {
		return dtoConfig{}, errors.New("invalid TOML syntax or value type")
	}

	return result, nil
}
