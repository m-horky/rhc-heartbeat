// Package chrony reads and interprets time synchronization data from chrony.
package chrony

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/m-horky/rhc-heartbeat/internal/command"
)

const trackingFieldCount = 13

// Tracking contains chrony's availability, synchronization state, and root distance.
type Tracking struct {
	Available            bool
	ClockSynchronized    bool
	ClockDistanceSeconds float64
}

// ReadTracking runs chronyc -c tracking and parses its CSV output. Command failures
// report unavailable tracking without failing, except when the context is canceled.
func ReadTracking(ctx context.Context, runner command.Runner) (Tracking, error) {
	result, err := runner.Run(ctx, "chronyc", "-c", "tracking")
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return Tracking{}, fmt.Errorf("run chronyc tracking: %w", contextErr)
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Tracking{}, fmt.Errorf("run chronyc tracking: %w", err)
		}

		return Tracking{}, nil
	}

	tracking, err := parseTracking(result.Stdout)
	if err != nil {
		return Tracking{}, fmt.Errorf("parse chronyc tracking output: %w", err)
	}

	return tracking, nil
}

// parseTracking parses the single CSV record returned by chronyc -c tracking.
func parseTracking(output []byte) (Tracking, error) {
	reader := csv.NewReader(strings.NewReader(string(output)))
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return Tracking{}, fmt.Errorf("read CSV: %w", err)
	}

	if len(records) != 1 {
		return Tracking{}, fmt.Errorf("expected one CSV record, got %d", len(records))
	}

	fields := records[0]
	if len(fields) != trackingFieldCount {
		return Tracking{}, fmt.Errorf("expected %d CSV fields, got %d", trackingFieldCount, len(fields))
	}

	rootDelay, err := parseFiniteFloat("root delay", fields[9])
	if err != nil {
		return Tracking{}, err
	}

	rootDispersion, err := parseFiniteFloat("root dispersion", fields[10])
	if err != nil {
		return Tracking{}, err
	}

	clockDistance := rootDispersion + rootDelay/2
	if math.IsInf(clockDistance, 0) || math.IsNaN(clockDistance) {
		return Tracking{}, errors.New("root distance is not finite")
	}

	synchronized, err := parseLeapStatus(fields[12])
	if err != nil {
		return Tracking{}, err
	}

	return Tracking{
		Available:            true,
		ClockSynchronized:    synchronized,
		ClockDistanceSeconds: clockDistance,
	}, nil
}

// parseFiniteFloat parses a finite floating-point CSV field with a descriptive error.
func parseFiniteFloat(name, value string) (float64, error) {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", name, err)
	}

	if math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		return 0, fmt.Errorf("parse %s: value must be finite", name)
	}

	return parsed, nil
}

// parseLeapStatus reports whether chrony indicates that the clock is synchronized.
func parseLeapStatus(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "normal", "insert second", "delete second":
		return true, nil
	case "not synchronised", "not synchronized":
		return false, nil
	default:
		return false, fmt.Errorf("unrecognized leap status %q", strings.TrimSpace(value))
	}
}
