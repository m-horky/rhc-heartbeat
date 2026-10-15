package run

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/m-horky/rhc-heartbeat/pkg/cache"
	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/consumer"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"github.com/m-horky/rhc-heartbeat/pkg/lock"
	"github.com/m-horky/rhc-heartbeat/pkg/upload"
)

// Run collects and uploads a heartbeat, retaining pending records when upload fails.
func Run(ctx context.Context, args []string) error {
	kind, err := parseKind(args)
	if errors.Is(err, flag.ErrHelp) {
		return printHelp(os.Stdout)
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
	hbCache, err := loadCache()
	if err != nil {
		return err
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

// loadCache loads pending heartbeats, rotating an unreadable cache file
// aside: the broken file is kept for investigation and the run continues
// with a fresh cache.
func loadCache() (*cache.Cache, error) {
	hbCache, err := cache.Load()
	if err != nil {
		rotated, rotateErr := cache.Rotate()
		if rotateErr != nil {
			return nil, fmt.Errorf("cannot rotate heartbeat cache: %w (cannot load cache: %w)", rotateErr, err)
		}

		slog.Error("cannot load heartbeat cache", "err", err, "rotated-to", rotated)

		if hbCache, err = cache.Load(); err != nil {
			return nil, fmt.Errorf("cannot load cache: %w", err)
		}
	}

	return hbCache, nil
}

// printHelp writes generated run-command flag help to output.
func printHelp(output io.Writer) error {
	var help strings.Builder

	flags := flag.NewFlagSet("heartbeat run", flag.ContinueOnError)
	flags.String("kind", string(heartbeat.KindPing), "kind of heartbeat to collect")

	_, _ = fmt.Fprintln(&help, "Usage: heartbeat run [flags]\n\nFlags:")
	flags.SetOutput(&help)
	flags.PrintDefaults()

	if _, err := io.WriteString(output, help.String()); err != nil {
		return fmt.Errorf("write run help: %w", err)
	}

	return nil
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
