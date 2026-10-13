package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	configcommand "github.com/m-horky/rhc-heartbeat/cmd/rhc-heartbeat/config"
	profilecommand "github.com/m-horky/rhc-heartbeat/cmd/rhc-heartbeat/profile"
	runcommand "github.com/m-horky/rhc-heartbeat/cmd/rhc-heartbeat/run"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
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

// run routes CLI actions.
func run(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		return printHelp(os.Stdout)
	}

	switch args[0] {
	case "config":
		if err := configcommand.Run(ctx, args[1:]); err != nil {
			return fmt.Errorf("config subcommand: %w", err)
		}

		return nil
	case "profile":
		if err := profilecommand.Run(ctx, args[1:]); err != nil {
			return fmt.Errorf("profile subcommand: %w", err)
		}

		return nil
	case "run":
		if err := runcommand.Run(ctx, args[1:]); err != nil {
			return fmt.Errorf("run subcommand: %w", err)
		}

		return nil
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

// printHelp writes generated command and flag help to output.
func printHelp(output io.Writer) error {
	var help strings.Builder

	flags := flag.NewFlagSet("heartbeat", flag.ContinueOnError)
	flags.String("kind", string(heartbeat.KindPing), "kind of heartbeat to collect")

	commands := []struct {
		name        string
		description string
	}{
		{name: "run", description: "collect and upload a heartbeat"},
		{name: "config", description: "print the resolved configuration"},
		{name: "profile", description: "print a collected profile as JSON"},
	}

	_, _ = fmt.Fprintln(&help, "Usage: heartbeat <command> [flags]")

	_, _ = fmt.Fprintln(&help, "\nCommands:")
	for _, command := range commands {
		_, _ = fmt.Fprintf(&help, "  %-10s %s\n", command.name, command.description)
	}

	_, _ = fmt.Fprintln(&help, "\nFlags:")
	flags.SetOutput(&help)
	flags.PrintDefaults()

	if _, err := io.WriteString(output, help.String()); err != nil {
		return fmt.Errorf("write command help: %w", err)
	}

	return nil
}
