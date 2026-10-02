package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/BurntSushi/toml"
	heartbeatconfig "github.com/m-horky/rhc-heartbeat/pkg/config"
)

// main runs the configuration inspection command and reports failures.
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	if err := run(); err != nil {
		slog.Error("command failed", "err", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run loads and writes the resolved heartbeat configuration.
func run() error {
	slog.Info("loading heartbeat configuration")

	cfg, err := heartbeatconfig.Get()
	if err != nil {
		return fmt.Errorf("load heartbeat configuration: %w", err)
	}

	encoder := toml.NewEncoder(os.Stdout)
	encoder.Indent = ""

	if err := encoder.Encode(redactProxyCredentials(cfg)); err != nil {
		return fmt.Errorf("write heartbeat configuration: %w", err)
	}

	slog.Info("resolved configuration output written")

	return nil
}

// redactProxyCredentials replaces configured proxy credentials before configuration is written to stdout.
func redactProxyCredentials(cfg heartbeatconfig.Config) heartbeatconfig.Config {
	if cfg.HTTP.Proxy.User != "" {
		cfg.HTTP.Proxy.User = "..."
	}

	if cfg.HTTP.Proxy.Password != "" {
		cfg.HTTP.Proxy.Password = "..."
	}

	return cfg
}
