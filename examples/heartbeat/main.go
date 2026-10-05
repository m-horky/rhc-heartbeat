package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

// main runs the heartbeat inspection command and reports failures.
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	if err := run(); err != nil {
		slog.Error("command failed", "err", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run collects a periodic heartbeat and writes it as indented JSON.
func run() error {
	hb, err := heartbeat.Get(context.Background(), heartbeat.KindPing)
	if err != nil {
		return fmt.Errorf("collect heartbeat: %w", err)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(hb); err != nil {
		return fmt.Errorf("write heartbeat: %w", err)
	}

	return nil
}
