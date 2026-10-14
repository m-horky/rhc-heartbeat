package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/m-horky/rhc-heartbeat/pkg/config"
)

// Run writes the resolved configuration as indented JSON with proxy credentials redacted.
func Run(_ context.Context, args []string) error {
	if len(args) != 0 {
		return errors.New("config subcommand does not accept arguments")
	}

	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("load heartbeat configuration: %w", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(redactProxyCredentials(cfg)); err != nil {
		return fmt.Errorf("write heartbeat configuration: %w", err)
	}

	return nil
}

// redactProxyCredentials replaces configured proxy credentials before configuration is written to stdout.
func redactProxyCredentials(cfg config.Config) config.Config {
	cfg.HTTP.Proxy.URI.User = nil

	return cfg
}
