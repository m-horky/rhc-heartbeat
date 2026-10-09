// Package main demonstrates collecting and uploading one heartbeat using Prometheus Remote Write.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"github.com/m-horky/rhc-heartbeat/pkg/upload"
)

// main runs the Remote Write example and reports failures.
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	if err := run(context.Background()); err != nil {
		slog.Error("heartbeat upload failed", "err", err)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run collects one periodic heartbeat and uploads it using the resolved configuration.
func run(ctx context.Context) error {
	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("load heartbeat configuration: %w", err)
	}

	hb, err := heartbeat.Get(ctx, heartbeat.KindPing)
	if err != nil {
		return fmt.Errorf("collect heartbeat: %w", err)
	}

	client, err := upload.New(cfg)
	if err != nil {
		return fmt.Errorf("create Prometheus Remote Write client: %w", err)
	}
	defer client.CloseIdleConnections()

	if err := client.Upload(ctx, []heartbeat.Heartbeat{hb}); err != nil {
		return fmt.Errorf("upload heartbeat: %w", err)
	}

	slog.Info("heartbeat uploaded", "endpoint", cfg.Heartbeat.URI.String())

	return nil
}
