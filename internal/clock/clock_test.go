package clock

import (
	"testing"

	"golang.org/x/sys/unix"
)

// TestReadReturnsTimespecs verifies all system clocks are returned without conversion.
//
// Given a running Linux system, when reading the clocks, then all timespec values are valid and current.
func TestReadReturnsTimespecs(t *testing.T) {
	t.Parallel()

	beforeMonotonic := readClockForTest(t, unix.CLOCK_MONOTONIC)
	beforeBoottime := readClockForTest(t, unix.CLOCK_BOOTTIME)
	beforeRealtime := readClockForTest(t, unix.CLOCK_REALTIME)

	got, err := Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	afterMonotonic := readClockForTest(t, unix.CLOCK_MONOTONIC)
	afterBoottime := readClockForTest(t, unix.CLOCK_BOOTTIME)
	afterRealtime := readClockForTest(t, unix.CLOCK_REALTIME)

	if !validTimespec(got.TimeMonotonic) {
		t.Errorf("TimeMonotonic = %+v, want valid timespec", got.TimeMonotonic)
	}

	if !validTimespec(got.TimeBoottime) {
		t.Errorf("TimeBoottime = %+v, want valid timespec", got.TimeBoottime)
	}

	if !validTimespec(got.Time) {
		t.Errorf("Time = %+v, want valid timespec", got.Time)
	}

	if timespecBefore(got.TimeMonotonic, beforeMonotonic) || timespecBefore(afterMonotonic, got.TimeMonotonic) {
		t.Errorf("TimeMonotonic = %+v, want value between %+v and %+v", got.TimeMonotonic, beforeMonotonic, afterMonotonic)
	}

	if timespecBefore(got.TimeBoottime, beforeBoottime) || timespecBefore(afterBoottime, got.TimeBoottime) {
		t.Errorf("TimeBoottime = %+v, want value between %+v and %+v", got.TimeBoottime, beforeBoottime, afterBoottime)
	}

	if timespecBefore(got.Time, beforeRealtime) || timespecBefore(afterRealtime, got.Time) {
		t.Errorf("Time = %+v, want value between %+v and %+v", got.Time, beforeRealtime, afterRealtime)
	}
}

// readClockForTest reads a clock timespec for test range comparisons.
func readClockForTest(t *testing.T, clockID int32) unix.Timespec {
	t.Helper()

	var reading unix.Timespec
	if err := unix.ClockGettime(clockID, &reading); err != nil {
		t.Fatalf("ClockGettime(%d) error = %v", clockID, err)
	}

	return reading
}

// validTimespec reports whether a timespec has a valid nonnegative second and nanosecond component.
func validTimespec(value unix.Timespec) bool {
	return value.Sec >= 0 && value.Nsec >= 0 && value.Nsec < 1_000_000_000
}

// timespecBefore reports whether left is earlier than right.
func timespecBefore(left, right unix.Timespec) bool {
	return left.Sec < right.Sec || (left.Sec == right.Sec && left.Nsec < right.Nsec)
}
