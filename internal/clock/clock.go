// Package clock reads raw monotonic uptime and wall-clock time from the operating system.
package clock

import (
	"fmt"
	"time"

	"golang.org/x/sys/unix"
)

// Reading contains the system's raw monotonic uptime and realtime clock reading.
type Reading struct {
	Uptime   time.Duration
	WallTime time.Time
}

// Read returns CLOCK_MONOTONIC_RAW as uptime and CLOCK_REALTIME as wall time.
func Read() (Reading, error) {
	var monotonic unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC_RAW, &monotonic); err != nil {
		return Reading{}, fmt.Errorf("read CLOCK_MONOTONIC_RAW: %w", err)
	}

	uptime, err := durationFromTimespec(monotonic.Sec, monotonic.Nsec)
	if err != nil {
		return Reading{}, fmt.Errorf("convert CLOCK_MONOTONIC_RAW reading: %w", err)
	}

	var realtime unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_REALTIME, &realtime); err != nil {
		return Reading{}, fmt.Errorf("read CLOCK_REALTIME: %w", err)
	}

	if realtime.Nsec < 0 || realtime.Nsec >= int64(time.Second) {
		return Reading{}, fmt.Errorf("convert CLOCK_REALTIME reading: nanoseconds out of range: %d", realtime.Nsec)
	}

	return Reading{
		Uptime:   uptime,
		WallTime: time.Unix(realtime.Sec, realtime.Nsec),
	}, nil
}

// durationFromTimespec converts a nonnegative timespec into a duration without overflow.
func durationFromTimespec(seconds, nanoseconds int64) (time.Duration, error) {
	if seconds < 0 {
		return 0, fmt.Errorf("seconds out of range: %d", seconds)
	}

	if nanoseconds < 0 || nanoseconds >= int64(time.Second) {
		return 0, fmt.Errorf("nanoseconds out of range: %d", nanoseconds)
	}

	const maxDuration = time.Duration(1<<63 - 1)

	maxSeconds := int64(maxDuration / time.Second)

	maxNanoseconds := int64(maxDuration % time.Second)
	if seconds > maxSeconds || (seconds == maxSeconds && nanoseconds > maxNanoseconds) {
		return 0, fmt.Errorf("timespec exceeds maximum time.Duration")
	}

	return time.Duration(seconds)*time.Second + time.Duration(nanoseconds), nil
}
