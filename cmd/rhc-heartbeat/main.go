// Package main provides the one-shot systemd entry point for collecting and uploading heartbeats.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/m-horky/rhc-heartbeat/pkg/cache"
	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/consumer"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"github.com/m-horky/rhc-heartbeat/pkg/lock"
	"github.com/m-horky/rhc-heartbeat/pkg/upload"
)

// main runs the command and exits unsuccessfully when heartbeat processing fails.
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))

	if err := execute(os.Args[1:]); err != nil {
		slog.Error("heartbeat command failed", "err", err)
		os.Exit(1)
	}
}

// execute runs the command with a context canceled by an interrupt or termination signal.
func execute(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return run(ctx, args)
}

// run collects a heartbeat, uploads pending records or retains failures.
func run(ctx context.Context, args []string) error {
	// Parse CLI input.
	kind, err := parseKind(args)
	if errors.Is(err, flag.ErrHelp) {
		_, _ = fmt.Fprintln(os.Stdout, "usage: rhc-heartbeat [--kind on|off|ping]")

		return nil
	}

	if err != nil {
		return err
	}

	// Ensure only one process instance runs at a time.
	flock, err := lock.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("cannot acquire application lock: %w", err)
	}
	defer func() {
		if err := flock.Close(); err != nil {
			slog.Debug("cannot release application lock", "err", err)
		}
	}()

	// Initialize configuration.
	cfg, err := config.Get()
	if err != nil {
		return fmt.Errorf("cannot load configuration: %w", err)
	}

	logInsecureTransport(cfg)

	// Load past heartbeats.
	hbCache, err := cache.Load()
	if err != nil {
		return fmt.Errorf("cannot load cache: %w", err)
	}

	// Collect current heartbeat.
	hb, err := heartbeat.Get(kind)
	if err != nil {
		return fmt.Errorf("cannot get heartbeat: %w", err)
	}
	hbCache.Add(hb)

	// Read identity.
	identity, err := consumer.Get()
	if err != nil {
		return handleFailure(hbCache, err)
	}

	// Include identity in current and all past heartbeats without one.
	hbCache.BindIdentity(identity)

	// Initialize the Prometheus client.
	uploader, err := upload.New(cfg)
	if err != nil {
		return handleFailure(hbCache, err)
	}
	defer uploader.CloseIdleConnections()

	// Attempt the upload.
	if err := uploader.Upload(ctx, hbCache.Read()); err != nil {
		return handleFailure(hbCache, err)
	}

	// Clean up.
	return handleSuccess(hbCache)
}

// logInsecureTransport warns when Remote Write traffic is not protected by HTTPS verification.
func logInsecureTransport(cfg config.Config) {
	if cfg.Heartbeat.URI.Scheme == "http" {
		slog.Warn("Remote Write endpoint uses plain HTTP; heartbeat data is not encrypted")
	}

	if cfg.Heartbeat.URI.Scheme == "https" && !cfg.Heartbeat.TLSVerify {
		slog.Warn("Remote Write TLS certificate verification is disabled")
	}
}

// handleSuccess clears the heartbeat cache after a successful upload.
func handleSuccess(hbCache *cache.Cache) error {
	slog.Info("upload complete")
	hbCache.Clear()

	if err := hbCache.Save(); err != nil {
		return fmt.Errorf("cannot clear heartbeat cache: %w", err)
	}

	return nil
}

// handleFailure saves pending heartbeats and returns the upload error.
func handleFailure(hbCache *cache.Cache, uploadErr error) error {
	if err := hbCache.Save(); err != nil {
		slog.Error("cannot save heartbeat cache", "err", err)

		return fmt.Errorf("save heartbeat cache: %w", err)
	}

	return fmt.Errorf("cannot upload heartbeat: %w", uploadErr)
}
