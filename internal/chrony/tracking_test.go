package chrony

import (
	"context"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/m-horky/rhc-heartbeat/internal/command"
)

// TestReadTrackingExecutesChronycAndParsesCSV verifies command invocation and tracking conversion.
//
// Given a runner that returns chronyc CSV, when reading tracking, then the command and parsed values are correct.
func TestReadTrackingExecutesChronycAndParsesCSV(t *testing.T) {
	t.Parallel()

	runner := &testRunner{result: command.Result{Stdout: []byte(trackingOutput("Normal", "0.06", "0.04"))}}

	got, err := ReadTracking(context.Background(), runner)
	if err != nil {
		t.Fatalf("ReadTracking() error = %v", err)
	}

	if runner.name != "chronyc" {
		t.Errorf("command name = %q, want %q", runner.name, "chronyc")
	}

	if !reflect.DeepEqual(runner.args, []string{"-c", "tracking"}) {
		t.Errorf("command args = %v, want %v", runner.args, []string{"-c", "tracking"})
	}

	if !got.Available {
		t.Errorf("Available = false, want true")
	}

	if !got.ClockSynchronized {
		t.Errorf("ClockSynchronized = false, want true")
	}

	if math.Abs(got.ClockDistanceSeconds-0.07) > 1e-12 {
		t.Errorf("ClockDistanceSeconds = %.12f, want 0.07", got.ClockDistanceSeconds)
	}
}

// TestReadTrackingMarksCommandUnavailable verifies command failures become unavailable tracking data.
//
// Given chronyc cannot execute or reach chronyd, when reading tracking,
// then availability is false without failing the read.
func TestReadTrackingMarksCommandUnavailable(t *testing.T) {
	t.Parallel()

	runner := &testRunner{err: errors.New("506 Cannot talk to daemon")}

	got, err := ReadTracking(context.Background(), runner)
	if err != nil {
		t.Fatalf("ReadTracking() error = %v, want nil for unavailable chrony", err)
	}

	if got.Available {
		t.Errorf("Available = true, want false")
	}
}

// TestReadTrackingPropagatesContextCancellation verifies cancellation is not treated as chrony unavailability.
//
// Given the command context is canceled, when reading tracking, then the cancellation error is returned.
func TestReadTrackingPropagatesContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runner := &testRunner{err: context.Canceled}

	_, err := ReadTracking(ctx, runner)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ReadTracking() error = %v, want context.Canceled", err)
	}
}

// TestParseTrackingValidatesFields verifies synchronization statuses and malformed chronyc records.
//
// Given valid or malformed chronyc CSV, when parsing tracking, then known statuses
// are mapped and invalid data is rejected.
func TestParseTrackingValidatesFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		output       string
		wantSync     bool
		wantDistance float64
		wantErr      bool
	}{
		{name: "normal", output: trackingOutput("Normal", "0.06", "0.04"), wantSync: true, wantDistance: 0.07},
		{
			name: "inserting leap second", output: trackingOutput("Insert second", "0", "0.02"),
			wantSync: true, wantDistance: 0.02,
		},
		{
			name: "deleting leap second", output: trackingOutput("Delete second", "0", "0.02"),
			wantSync: true, wantDistance: 0.02,
		},
		{name: "not synchronized", output: trackingOutput("Not synchronised", "0", "0.02"), wantDistance: 0.02},
		{name: "unknown leap status", output: trackingOutput("Unknown", "0", "0.02"), wantErr: true},
		{name: "invalid root delay", output: trackingOutput("Normal", "bad", "0.02"), wantErr: true},
		{name: "non-finite root dispersion", output: trackingOutput("Normal", "0", "NaN"), wantErr: true},
		{name: "missing field", output: "ref,2,time,0,0,0,0,0,0,0,0,64\n", wantErr: true},
		{
			name:    "multiple records",
			output:  trackingOutput("Normal", "0", "0.02") + trackingOutput("Normal", "0", "0.02"),
			wantErr: true,
		},
		{name: "malformed CSV", output: "\"unterminated\n", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseTracking([]byte(test.output))
			if (err != nil) != test.wantErr {
				t.Fatalf("parseTracking() error = %v, wantErr %t", err, test.wantErr)
			}

			if err != nil {
				return
			}

			if got.ClockSynchronized != test.wantSync {
				t.Errorf("ClockSynchronized = %t, want %t", got.ClockSynchronized, test.wantSync)
			}

			if math.Abs(got.ClockDistanceSeconds-test.wantDistance) > 1e-12 {
				t.Errorf("ClockDistanceSeconds = %.12f, want %.12f", got.ClockDistanceSeconds, test.wantDistance)
			}
		})
	}
}

// trackingOutput builds one chronyc -c tracking CSV record for tests.
func trackingOutput(leapStatus, rootDelay, rootDispersion string) string {
	fields := []string{
		"ref-id", "2", "2025-03-01T12:00:00Z", "0", "0", "0", "0", "0", "0",
		rootDelay, rootDispersion, "64", leapStatus,
	}

	return strings.Join(fields, ",") + "\n"
}

type testRunner struct {
	name   string
	args   []string
	result command.Result
	err    error
}

// Run records the process request and returns the configured test result.
func (runner *testRunner) Run(_ context.Context, name string, args ...string) (command.Result, error) {
	runner.name = name

	runner.args = append([]string(nil), args...)

	return runner.result, runner.err
}
