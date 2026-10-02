package clock

import (
	"math"
	"testing"
	"time"
)

// TestReadReturnsSystemClocks verifies both requested system clocks are read.
//
// Given a running Linux system, when reading the clocks, then uptime is nonnegative and wall time is current.
func TestReadReturnsSystemClocks(t *testing.T) {
	t.Parallel()

	before := time.Now()
	got, err := Read()
	after := time.Now()

	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if got.Uptime < 0 {
		t.Errorf("Read().Uptime = %s, want a nonnegative monotonic uptime", got.Uptime)
	}

	if got.WallTime.Before(before) || got.WallTime.After(after) {
		t.Errorf("Read().WallTime = %s, want a time between %s and %s", got.WallTime, before, after)
	}
}

// TestDurationFromTimespecValidatesInput verifies timespec values convert safely to durations.
//
// Given a valid or malformed timespec, when converting it, then it is converted or rejected safely.
func TestDurationFromTimespecValidatesInput(t *testing.T) {
	t.Parallel()

	const maxDuration = time.Duration(math.MaxInt64)

	tests := []struct {
		name        string
		seconds     int64
		nanoseconds int64
		want        time.Duration
		wantErr     bool
	}{
		{name: "whole seconds", seconds: 3, nanoseconds: 4, want: 3*time.Second + 4},
		{
			name:        "maximum duration",
			seconds:     int64(maxDuration / time.Second),
			nanoseconds: int64(maxDuration % time.Second),
			want:        maxDuration,
		},
		{name: "negative seconds", seconds: -1, wantErr: true},
		{name: "negative nanoseconds", nanoseconds: -1, wantErr: true},
		{name: "nanoseconds out of range", nanoseconds: int64(time.Second), wantErr: true},
		{name: "duration overflow", seconds: int64(maxDuration/time.Second) + 1, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := durationFromTimespec(test.seconds, test.nanoseconds)
			if (err != nil) != test.wantErr {
				t.Fatalf("durationFromTimespec() error = %v, wantErr %t", err, test.wantErr)
			}

			if err == nil && got != test.want {
				t.Errorf("durationFromTimespec() = %s, want %s", got, test.want)
			}
		})
	}
}
