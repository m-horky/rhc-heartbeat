// Package clock reads monotonic uptime, boot time, and wall-clock time from the operating system.
package clock

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// Reading contains time values for the system clocks.
type Reading struct {
	TimeMonotonic unix.Timespec
	TimeUnix      unix.Timespec
}

// Read returns CLOCK_MONOTONIC and CLOCK_REALTIME as unix.Timespec values.
func Read() (Reading, error) {
	var monotonic unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC, &monotonic); err != nil {
		return Reading{}, fmt.Errorf("read CLOCK_MONOTONIC: %w", err)
	}

	var realtime unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_REALTIME, &realtime); err != nil {
		return Reading{}, fmt.Errorf("read CLOCK_REALTIME: %w", err)
	}

	return Reading{
		TimeMonotonic: monotonic,
		TimeUnix:      realtime,
	}, nil
}
