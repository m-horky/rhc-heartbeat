// Package main demonstrates collecting and exporting one heartbeat over OTLP/HTTP.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/m-horky/rhc-heartbeat/internal/otlp"
	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

// main runs the heartbeat export example and reports failures.
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	if err := run(context.Background()); err != nil {
		slog.Error("heartbeat export failed", "err", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run collects one periodic heartbeat and exports it using the resolved configuration.
func run(ctx context.Context) error {
	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("load heartbeat configuration: %w", err)
	}

	hb, err := heartbeat.Get(ctx, heartbeat.TriggerPing)
	if err != nil {
		return fmt.Errorf("collect heartbeat: %w", err)
	}

	client, err := otlp.New(cfg)
	if err != nil {
		return fmt.Errorf("create OTLP client: %w", err)
	}
	defer client.CloseIdleConnections()

	if err := client.Upload(ctx, []heartbeat.Heartbeat{hb}); err != nil {
		return fmt.Errorf("upload heartbeat: %w", err)
	}

	slog.Info("heartbeat exported", "endpoint", cfg.OTEL.URI)

	return nil
}
