// Package heartbeat collects the system state represented by a heartbeat.
package heartbeat

import (
	"context"
	"fmt"
	"time"

	"github.com/m-horky/rhc-heartbeat/internal/bootid"
	"github.com/m-horky/rhc-heartbeat/internal/clock"
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

// Heartbeat contains the system identity and time data collected for one event.
// TimeMonotonic is CLOCK_MONOTONIC elapsed time since boot, TimeBoottime is
// CLOCK_BOOTTIME elapsed time since boot including suspend, and TimeUnix is the
// CLOCK_REALTIME timestamp of the event. All preserve nanosecond resolution.
type Heartbeat struct {
	HostID        string
	HostOrg       string
	BootID        string
	TimeMonotonic time.Duration
	TimeBoottime  time.Duration
	TimeUnix      time.Time
	Trigger       Trigger
}

type sources struct {
	readIdentity func() (consumer.Identity, error)
	readBootID   func() (string, error)
	readClock    func() (clock.Reading, error)
}

// Get collects a heartbeat using the system defaults and the requested trigger.
func Get(ctx context.Context, trigger Trigger) (Heartbeat, error) {
	return collect(ctx, trigger, sources{
		readIdentity: consumer.Get,
		readBootID: func() (string, error) {
			return bootid.Read(fs.Filesystem{})
		},
		readClock: clock.Read,
	})
}

// collect reads each heartbeat source and combines the results into one value.
func collect(ctx context.Context, trigger Trigger, source sources) (Heartbeat, error) {
	if trigger != TriggerPing && trigger != TriggerOff {
		return Heartbeat{}, fmt.Errorf("invalid heartbeat trigger %q", trigger)
	}

	if err := ctx.Err(); err != nil {
		return Heartbeat{}, fmt.Errorf("collect heartbeat: %w", err)
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

	monotonicTime := time.Duration(clockReading.TimeMonotonic.Sec)*time.Second +
		time.Duration(clockReading.TimeMonotonic.Nsec)
	boottime := time.Duration(clockReading.TimeBoottime.Sec)*time.Second +
		time.Duration(clockReading.TimeBoottime.Nsec)
	unixTime := time.Unix(clockReading.Time.Sec, clockReading.Time.Nsec).UTC()

	return Heartbeat{
		HostID:        identity.UUID,
		HostOrg:       identity.OrgID,
		BootID:        bootID,
		TimeMonotonic: monotonicTime,
		TimeBoottime:  boottime,
		TimeUnix:      unixTime,
		Trigger:       trigger,
	}, nil
}
