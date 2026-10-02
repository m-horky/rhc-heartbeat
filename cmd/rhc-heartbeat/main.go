// Package main provides the one-shot systemd entry point for collecting and uploading heartbeats.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/m-horky/rhc-heartbeat/internal/constants"
	"github.com/m-horky/rhc-heartbeat/pkg/cache"
	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"github.com/m-horky/rhc-heartbeat/pkg/process"
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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return run(ctx, args)
}

// run collects one heartbeat, loads configuration, and processes it with the pending cache.
func run(ctx context.Context, args []string) error {
	trigger, err := parseTrigger(args)
	if errors.Is(err, flag.ErrHelp) {
		writeUsage(os.Stdout)

		return nil
	}

	if err != nil {
		return err
	}

	pending := cache.New(constants.PathFromEnv(constants.PendingCachePathEnv, constants.DefaultPendingCachePath))

	hb, err := heartbeat.Get(ctx, trigger)
	if err != nil {
		return fmt.Errorf("collect heartbeat: %w", err)
	}

	cfg, err := config.Get()
	if err != nil {
		return cacheAfterSetupFailure(pending, hb, fmt.Errorf("load heartbeat configuration: %w", err))
	}

	processor, err := process.New(cfg, pending)
	if err != nil {
		return cacheAfterSetupFailure(pending, hb, fmt.Errorf("create heartbeat processor: %w", err))
	}
	defer processor.CloseIdleConnections()

	if err := processor.Process(ctx, hb); err != nil {
		return fmt.Errorf("process heartbeat: %w", err)
	}

	slog.Info("heartbeat processing completed", "trigger", trigger, "endpoint", cfg.OTEL.URI)

	return nil
}

// parseTrigger parses command flags and validates the requested heartbeat trigger.
func parseTrigger(args []string) (heartbeat.Trigger, error) {
	flags := flag.NewFlagSet("rhc-heartbeat", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	triggerValue := flags.String("trigger", string(heartbeat.TriggerPing), "heartbeat trigger: ping or off")

	if err := flags.Parse(args); err != nil {
		return "", fmt.Errorf("parse command flags: %w", err)
	}

	if flags.NArg() > 0 {
		return "", fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}

	trigger := heartbeat.Trigger(*triggerValue)
	if trigger != heartbeat.TriggerPing && trigger != heartbeat.TriggerOff {
		return "", fmt.Errorf("invalid heartbeat trigger %q: want ping or off", trigger)
	}

	return trigger, nil
}

// cacheAfterSetupFailure retains a collected heartbeat when configuration or processor setup fails.
func cacheAfterSetupFailure(pending *cache.Cache, hb heartbeat.Heartbeat, setupErr error) error {
	slog.Warn("heartbeat setup failed; caching for retry", "err", setupErr)

	if err := pending.Append(hb); err != nil {
		return errors.Join(setupErr, fmt.Errorf("cache heartbeat after setup failure: %w", err))
	}

	return nil
}

// writeUsage writes the supported command syntax to the supplied writer.
func writeUsage(writer io.Writer) {
	_, _ = fmt.Fprintln(writer, "Usage: rhc-heartbeat [--trigger ping|off]")
	_, _ = fmt.Fprintln(writer, "Collect and upload one heartbeat, then backfill pending records.")
}
