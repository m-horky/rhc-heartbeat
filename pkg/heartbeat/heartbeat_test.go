package heartbeat

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/m-horky/rhc-heartbeat/internal/clock"
	"github.com/m-horky/rhc-heartbeat/pkg/consumer"
	"golang.org/x/sys/unix"
)

// TestCollectAssemblesHeartbeat verifies successful source readings populate the heartbeat.
//
// Given successful identity, boot ID, and clock sources, when collecting,
// then all readings and the trigger are returned.
func TestCollectAssemblesHeartbeat(t *testing.T) {
	t.Parallel()

	monotonic := unix.Timespec{Sec: 42, Nsec: 123}
	boottime := unix.Timespec{Sec: 45, Nsec: 789}
	realtime := unix.Timespec{Sec: 1_741_000_000, Nsec: 456}
	source := testSources()
	source.readIdentity = func() (consumer.Identity, error) {
		return consumer.Identity{UUID: "system-uuid", OrgID: "org-id"}, nil
	}
	source.readBootID = func() (string, error) { return "boot-id", nil }
	source.readClock = func() (clock.Reading, error) {
		return clock.Reading{TimeMonotonic: monotonic, TimeBoottime: boottime, Time: realtime}, nil
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

	wantMonotonic := time.Duration(monotonic.Sec)*time.Second + time.Duration(monotonic.Nsec)
	if got.TimeMonotonic != wantMonotonic {
		t.Errorf("TimeMonotonic = %s, want %s", got.TimeMonotonic, wantMonotonic)
	}

	wantBoottime := time.Duration(boottime.Sec)*time.Second + time.Duration(boottime.Nsec)
	if got.TimeBoottime != wantBoottime {
		t.Errorf("TimeBoottime = %s, want %s", got.TimeBoottime, wantBoottime)
	}

	wantUnix := time.Unix(realtime.Sec, realtime.Nsec).UTC()
	if !got.TimeUnix.Equal(wantUnix) {
		t.Errorf("TimeUnix = %s, want %s", got.TimeUnix, wantUnix)
	}

	if got.Trigger != TriggerOff {
		t.Errorf("Trigger = %q, want %q", got.Trigger, TriggerOff)
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

// TestCollectPropagatesContextCancellation verifies canceled collection contexts are respected.
//
// Given an already canceled context, when collecting a heartbeat, then the cancellation is returned.
func TestCollectPropagatesContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := collect(ctx, TriggerPing, testSources())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("collect() error = %v, want context.Canceled", err)
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

			source := testSources()
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

// testSources returns successful heartbeat sources for collection tests.
func testSources() sources {
	return sources{
		readIdentity: func() (consumer.Identity, error) {
			return consumer.Identity{UUID: "test-uuid", OrgID: "test-org"}, nil
		},
		readBootID: func() (string, error) {
			return "test-boot-id", nil
		},
		readClock: func() (clock.Reading, error) {
			return clock.Reading{
				TimeMonotonic: unix.Timespec{Sec: 1, Nsec: 2},
				TimeBoottime:  unix.Timespec{Sec: 1, Nsec: 4},
				Time:          unix.Timespec{Sec: 1, Nsec: 3},
			}, nil
		},
	}
}
