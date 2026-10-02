// Package heartbeat collects the system state represented by a heartbeat.
package heartbeat

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/m-horky/rhc-heartbeat/internal/bootid"
	"github.com/m-horky/rhc-heartbeat/internal/chrony"
	"github.com/m-horky/rhc-heartbeat/internal/clock"
	"github.com/m-horky/rhc-heartbeat/internal/command"
	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/consumer"
)

// Trigger describes why a heartbeat was collected.
type Trigger string

const (
	// TriggerPing identifies a periodic heartbeat.
	TriggerPing Trigger = "ping"
	// TriggerOff identifies a heartbeat collected during shutdown.
	TriggerOff Trigger = "off"
)

// TimeQuality describes whether chrony reports the system clock synchronized.
type TimeQuality string

const (
	// TimeQualityUnknown means time quality could not be collected.
	TimeQualityUnknown TimeQuality = "unknown"
	// TimeQualityDesync means chrony reports that the clock is not synchronized.
	TimeQualityDesync TimeQuality = "desync"
)

// Heartbeat contains the system identity and time data collected for one event.
// TimeMonotonic is CLOCK_MONOTONIC_RAW elapsed time since boot, and TimeUnix is
// the CLOCK_REALTIME timestamp of the event. Both preserve nanosecond resolution.
type Heartbeat struct {
	HostID        string
	HostOrg       string
	BootID        string
	TimeMonotonic time.Duration
	TimeUnix      time.Time
	TimeQuality   TimeQuality
	Trigger       Trigger
}

type sources struct {
	readIdentity func() (consumer.Identity, error)
	readBootID   func() (string, error)
	readClock    func() (clock.Reading, error)
	runner       command.Runner
}

// Get collects a heartbeat using the system defaults and the requested trigger.
func Get(ctx context.Context, trigger Trigger) (Heartbeat, error) {
	return collect(ctx, trigger, sources{
		readIdentity: consumer.Get,
		readBootID: func() (string, error) {
			return bootid.Read(fs.Filesystem{})
		},
		readClock: clock.Read,
		runner:    command.OSRunner{},
	})
}

// collect reads each heartbeat source and combines the results into one value.
func collect(ctx context.Context, trigger Trigger, source sources) (Heartbeat, error) {
	if trigger != TriggerPing && trigger != TriggerOff {
		return Heartbeat{}, fmt.Errorf("invalid heartbeat trigger %q", trigger)
	}

	identity, err := source.readIdentity()
	if err != nil {
		return Heartbeat{}, fmt.Errorf("read heartbeat identity: %w", err)
	}

	bootID, err := source.readBootID()
	if err != nil {
		return Heartbeat{}, fmt.Errorf("read heartbeat boot ID: %w", err)
	}

	clockReading, err := source.readClock()
	if err != nil {
		return Heartbeat{}, fmt.Errorf("read heartbeat clocks: %w", err)
	}

	timeQuality := TimeQualityUnknown

	tracking, err := chrony.ReadTracking(ctx, source.runner)
	if err != nil {
		if contextErr := ctx.Err(); contextErr != nil {
			return Heartbeat{}, fmt.Errorf("read heartbeat chrony status: %w", contextErr)
		}

		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Heartbeat{}, fmt.Errorf("read heartbeat chrony status: %w", err)
		}
	} else {
		timeQuality = timeQualityFromTracking(tracking)
	}

	monotonicTime := time.Duration(clockReading.TimeMonotonic.Sec)*time.Second +
		time.Duration(clockReading.TimeMonotonic.Nsec)
	unixTime := time.Unix(clockReading.Time.Sec, clockReading.Time.Nsec).UTC()

	return Heartbeat{
		HostID:        identity.UUID,
		HostOrg:       identity.OrgID,
		BootID:        bootID,
		TimeMonotonic: monotonicTime,
		TimeUnix:      unixTime,
		TimeQuality:   timeQuality,
		Trigger:       trigger,
	}, nil
}

// timeQualityFromTracking maps chrony tracking data into the heartbeat quality value.
func timeQualityFromTracking(tracking chrony.Tracking) TimeQuality {
	if !tracking.Available {
		return TimeQualityUnknown
	}

	if !tracking.ClockSynchronized {
		return TimeQualityDesync
	}

	return TimeQuality("sync:" + strconv.FormatFloat(tracking.ClockDistanceSeconds, 'f', -1, 64))
}
