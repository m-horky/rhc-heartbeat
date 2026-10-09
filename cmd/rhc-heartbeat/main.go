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
	"github.com/m-horky/rhc-heartbeat/pkg/constants"
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

// run collects one heartbeat, configures delivery, uploads it with pending records, and caches failures.
func run(ctx context.Context, args []string) error {
	// Parse input first.
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

	// Initialize Prometheus client.
	uploader, err := upload.New(cfg)
	if err != nil {
		return fmt.Errorf("cannot create Prometheus Remote Write client: %w", err)
	}
	defer uploader.CloseIdleConnections()

	// Load past heartbeats.
	hbCache, err := cache.Load(constants.PathFromEnv(constants.PendingCachePathEnv, constants.DefaultPendingCachePath))
	if err != nil {
		return fmt.Errorf("cannot load cache: %w", err)
	}

	// Collect current heartbeat.
	hb, err := heartbeat.Get(ctx, kind)
	if err != nil {
		return fmt.Errorf("cannot collect heartbeat: %w", err)
	}

	hbCache.Add(hb)

	// Attempt the upload.
	if err := uploader.Upload(ctx, hbCache.Read()); err == nil {
		return handleUploadSuccess(hbCache)
	} else {
		return handleUploadFailure(hbCache, err)
	}
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

// handleUploadSuccess clears the heartbeat cache after a successful upload.
func handleUploadSuccess(hbCache *cache.Cache) error {
	slog.Info("upload complete")
	hbCache.Clear()

	if err := hbCache.Save(); err != nil {
		return fmt.Errorf("cannot clear heartbeat cache: %w", err)
	}

	return nil
}

// handleUploadFailure saves pending heartbeats and returns the upload error.
func handleUploadFailure(hbCache *cache.Cache, uploadErr error) error {
	slog.Error("heartbeat upload failed", "err", uploadErr)

	if err := hbCache.Save(); err == nil {
		slog.Info("heartbeat cache saved")
	} else {
		slog.Warn("cannot save heartbeat cache", "err", err)
	}

	return fmt.Errorf("cannot upload heartbeat: %w", uploadErr)
}
