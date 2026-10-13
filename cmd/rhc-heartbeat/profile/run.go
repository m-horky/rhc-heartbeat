package profile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

// Run collects a periodic heartbeat and writes it as indented JSON.
func Run(_ context.Context, args []string) error {
	if len(args) != 0 {
		return errors.New("profile subcommand does not accept arguments")
	}

	hb, err := heartbeat.Get(heartbeat.KindPing)
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
