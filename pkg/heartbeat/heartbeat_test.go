package heartbeat

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/m-horky/rhc-heartbeat/internal/chrony"
	"github.com/m-horky/rhc-heartbeat/internal/clock"
	"github.com/m-horky/rhc-heartbeat/internal/command"
	"github.com/m-horky/rhc-heartbeat/pkg/consumer"
	"golang.org/x/sys/unix"
)

// TestCollectAssemblesHeartbeat verifies all source readings populate the public heartbeat.
//
// Given successful identity, boot ID, clock, and chrony sources, when collecting,
// then all readings and the trigger are returned.
func TestCollectAssemblesHeartbeat(t *testing.T) {
	t.Parallel()

	monotonic := unix.Timespec{Sec: 42, Nsec: 123}
	realtime := unix.Timespec{Sec: 1_741_000_000, Nsec: 456}
	runner := &testRunner{result: command.Result{Stdout: []byte(validChronyCSV())}}
	source := testSources(runner)
	source.readIdentity = func() (consumer.Identity, error) {
		return consumer.Identity{UUID: "system-uuid", OrgID: "org-id"}, nil
	}
	source.readBootID = func() (string, error) { return "boot-id", nil }
	source.readClock = func() (clock.Reading, error) {
		return clock.Reading{TimeMonotonic: monotonic, Time: realtime}, nil
	}

	got, err := collect(context.Background(), TriggerOff, source)
	if err != nil {
		t.Fatalf("collect() error = %v", err)
	}

	if got.HostID != "system-uuid" {
		t.Errorf("HostID = %q, want %q", got.HostID, "system-uuid")
	}

	if got.HostOrg != "org-id" {
		t.Errorf("HostOrg = %q, want %q", got.HostOrg, "org-id")
	}

	if got.BootID != "boot-id" {
		t.Errorf("BootID = %q, want %q", got.BootID, "boot-id")
	}

	if got.TimeMonotonic != monotonic.Sec {
		t.Errorf("TimeMonotonic = %d, want %d seconds", got.TimeMonotonic, monotonic.Sec)
	}

	if got.TimeUnix != realtime.Sec {
		t.Errorf("TimeUnix = %d, want %d seconds", got.TimeUnix, realtime.Sec)
	}

	if got.TimeQuality != TimeQuality("sync:0.07") {
		t.Errorf("TimeQuality = %q, want %q", got.TimeQuality, "sync:0.07")
	}

	if got.Trigger != TriggerOff {
		t.Errorf("Trigger = %q, want %q", got.Trigger, TriggerOff)
	}

	if runner.name != "chronyc" || strings.Join(runner.args, " ") != "-c tracking" {
		t.Errorf("command = %s %s, want chronyc -c tracking", runner.name, strings.Join(runner.args, " "))
	}
}

// TestCollectContinuesWhenChronyIsUnavailable verifies missing chrony data does not block heartbeat collection.
//
// Given the chrony command fails, when collecting, then the heartbeat is returned with ChronyAvailable false.
func TestCollectContinuesWhenChronyIsUnavailable(t *testing.T) {
	t.Parallel()

	source := testSources(&testRunner{err: errors.New("506 Cannot talk to daemon")})

	got, err := collect(context.Background(), TriggerPing, source)
	if err != nil {
		t.Fatalf("collect() error = %v, want nil for unavailable chrony", err)
	}

	if got.TimeQuality != TimeQualityUnknown {
		t.Errorf("TimeQuality = %q, want %q", got.TimeQuality, TimeQualityUnknown)
	}
}

// TestCollectUsesUnknownWhenChronyOutputCannotBeParsed verifies chrony collection errors map to unknown quality.
//
// Given chrony returns malformed output, when collecting, then the heartbeat is returned with unknown quality.
func TestCollectUsesUnknownWhenChronyOutputCannotBeParsed(t *testing.T) {
	t.Parallel()

	source := testSources(&testRunner{result: command.Result{Stdout: []byte("invalid output")}})

	got, err := collect(context.Background(), TriggerPing, source)
	if err != nil {
		t.Fatalf("collect() error = %v, want nil when time quality is unknown", err)
	}

	if got.TimeQuality != TimeQualityUnknown {
		t.Errorf("TimeQuality = %q, want %q", got.TimeQuality, TimeQualityUnknown)
	}
}

// TestCollectPropagatesChronyContextCancellation verifies heartbeat collection respects cancellation.
//
// Given the chrony command context is canceled, when collecting, then the cancellation error is returned.
func TestCollectPropagatesChronyContextCancellation(t *testing.T) {
	t.Parallel()

	source := testSources(&testRunner{err: context.Canceled})

	_, err := collect(context.Background(), TriggerPing, source)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("collect() error = %v, want context.Canceled", err)
	}
}

// TestTimeQualityFromTrackingMapsAvailableStates verifies chrony data maps to the quality values.
//
// Given unavailable, unsynchronized, or synchronized chrony data, when mapping quality,
// then the expected value is returned.
func TestTimeQualityFromTrackingMapsAvailableStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tracking chrony.Tracking
		want     TimeQuality
	}{
		{name: "unavailable", tracking: chrony.Tracking{}, want: TimeQualityUnknown},
		{name: "unsynchronized", tracking: chrony.Tracking{Available: true}, want: TimeQualityDesync},
		{
			name: "synchronized",
			tracking: chrony.Tracking{
				Available: true, ClockSynchronized: true, ClockDistanceSeconds: 0.125,
			},
			want: TimeQuality("sync:0.125"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := timeQualityFromTracking(test.tracking); got != test.want {
				t.Errorf("timeQualityFromTracking() = %q, want %q", got, test.want)
			}
		})
	}
}

// TestCollectRejectsUnknownTrigger verifies invalid trigger values are rejected before reading sources.
//
// Given an unsupported trigger and no configured sources, when collecting,
// then validation fails without reading any source.
func TestCollectRejectsUnknownTrigger(t *testing.T) {
	t.Parallel()

	_, err := collect(context.Background(), Trigger("unknown"), sources{})
	if err == nil || !strings.Contains(err.Error(), "invalid heartbeat trigger") {
		t.Fatalf("collect() error = %v, want invalid trigger error", err)
	}
}

// TestCollectWrapsSourceErrors verifies collection stops and identifies the failing source.
//
// Given any heartbeat source fails, when collecting, then its failure is returned with source context.
func TestCollectWrapsSourceErrors(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("source unavailable")
	tests := []struct {
		name        string
		configure   func(*sources)
		wantContext string
	}{
		{
			name: "identity",
			configure: func(source *sources) {
				source.readIdentity = func() (consumer.Identity, error) { return consumer.Identity{}, wantErr }
			},
			wantContext: "read heartbeat identity",
		},
		{
			name: "boot ID",
			configure: func(source *sources) {
				source.readBootID = func() (string, error) { return "", wantErr }
			},
			wantContext: "read heartbeat boot ID",
		},
		{
			name: "clock",
			configure: func(source *sources) {
				source.readClock = func() (clock.Reading, error) { return clock.Reading{}, wantErr }
			},
			wantContext: "read heartbeat clocks",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			source := testSources(&testRunner{result: command.Result{Stdout: []byte(validChronyCSV())}})
			test.configure(&source)

			_, err := collect(context.Background(), TriggerPing, source)
			if !errors.Is(err, wantErr) {
				t.Fatalf("collect() error = %v, want wrapped %v", err, wantErr)
			}

			if !strings.Contains(err.Error(), test.wantContext) {
				t.Errorf("collect() error = %v, want context %q", err, test.wantContext)
			}
		})
	}
}

// testSources returns successful heartbeat sources backed by the supplied command runner.
func testSources(runner command.Runner) sources {
	return sources{
		readIdentity: func() (consumer.Identity, error) {
			return consumer.Identity{UUID: "test-uuid", OrgID: "test-org"}, nil
		},
		readBootID: func() (string, error) {
			return "test-boot-id", nil
		},
		readClock: func() (clock.Reading, error) {
			return clock.Reading{
				TimeMonotonic: unix.Timespec{Sec: 1},
				Time:          unix.Timespec{Sec: 1},
			}, nil
		},
		runner: runner,
	}
}

// validChronyCSV returns a valid chronyc -c tracking record for tests.
func validChronyCSV() string {
	return "ref-id,2,2025-03-01T12:00:00Z,0,0,0,0,0,0,0.06,0.04,64,Normal\n"
}

type testRunner struct {
	name   string
	args   []string
	result command.Result
	err    error
}

// Run records the command request and returns the configured test result.
func (runner *testRunner) Run(_ context.Context, name string, args ...string) (command.Result, error) {
	runner.name = name

	runner.args = append([]string(nil), args...)

	return runner.result, runner.err
}
