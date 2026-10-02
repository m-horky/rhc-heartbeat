// Package clock reads raw monotonic uptime and wall-clock time from the operating system.
package clock

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// Reading contains the unconverted timespec values for the system clocks.
type Reading struct {
	TimeMonotonic unix.Timespec
	Time          unix.Timespec
}

// Read returns CLOCK_MONOTONIC_RAW and CLOCK_REALTIME as unix.Timespec values.
func Read() (Reading, error) {
	var monotonic unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC_RAW, &monotonic); err != nil {
		return Reading{}, fmt.Errorf("read CLOCK_MONOTONIC_RAW: %w", err)
	}

	var realtime unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_REALTIME, &realtime); err != nil {
		return Reading{}, fmt.Errorf("read CLOCK_REALTIME: %w", err)
	}

	return Reading{
		TimeMonotonic: monotonic,
		Time:          realtime,
	}, nil
}
