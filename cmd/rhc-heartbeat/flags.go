package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

// parseKind parses and validates the heartbeat kind supplied on the command line.
func parseKind(args []string) (heartbeat.Kind, error) {
	flags := flag.NewFlagSet("rhc-heartbeat", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	kindValue := flags.String("kind", string(heartbeat.KindPing), "kind of heartbeat to collect")

	if err := flags.Parse(args); err != nil {
		return "", fmt.Errorf("parse command flags: %w", err)
	}

	if flags.NArg() > 0 {
		return "", fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}

	kind := heartbeat.Kind(*kindValue)
	if !kind.Valid() {
		return "", fmt.Errorf("invalid --kind %q: must be one of %q, %q, or %q", kind,
			heartbeat.KindOn, heartbeat.KindOff, heartbeat.KindPing)
	}

	return kind, nil
}
