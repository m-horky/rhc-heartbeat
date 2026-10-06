package lock

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestAcquireWaitsUntilLockReleased verifies an acquisition blocks until the current owner releases its lock.
//
// Given one process holds the lock, when another process requests it and the owner releases it,
// then the waiting process acquires the lock.
func TestAcquireWaitsUntilLockReleased(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "heartbeat.lock")

	owner, err := Acquire(context.Background(), path, nil)
	if err != nil {
		t.Fatalf("Acquire() owner error = %v", err)
	}

	result := make(chan struct {
		lock *Lock
		err  error
	}, 1)
	go func() {
		lock, err := Acquire(context.Background(), path, nil)
		result <- struct {
			lock *Lock
			err  error
		}{lock: lock, err: err}
	}()

	// Allow the waiting goroutine to observe the held lock before releasing it.
	time.Sleep(50 * time.Millisecond)

	select {
	case acquired := <-result:
		if acquired.lock != nil {
			_ = acquired.lock.Close()
		}

		t.Fatalf("Acquire() while lock is held returned error %v, want it to wait", acquired.err)
	case <-time.After(50 * time.Millisecond):
	}

	if err := owner.Close(); err != nil {
		t.Fatalf("owner.Close() error = %v", err)
	}

	select {
	case acquired := <-result:
		if acquired.err != nil {
			t.Fatalf("waiting Acquire() error = %v", acquired.err)
		}

		if err := acquired.lock.Close(); err != nil {
			t.Fatalf("waiting lock.Close() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiting Acquire() did not acquire the released lock")
	}
}

// TestAcquireCallsOnWait verifies the callback receives the current owner PID once on contention.
//
// Given another process holds the lock, when a request waits for it, then the callback receives its PID.
func TestAcquireCallsOnWait(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "heartbeat.lock")

	owner, err := Acquire(context.Background(), path, nil)
	if err != nil {
		t.Fatalf("Acquire() owner error = %v", err)
	}

	defer func() {
		if err := owner.Close(); err != nil {
			t.Errorf("owner.Close() error = %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	called := false

	_, err = Acquire(ctx, path, func(pid int) {
		called = true

		if pid <= 0 {
			t.Errorf("onWait PID = %d, want positive PID", pid)
		}
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Acquire() error = %v, want context deadline exceeded", err)
	}

	if !called {
		t.Fatal("Acquire() did not call onWait while lock was held")
	}
}

// TestAcquireWaitHonorsContext verifies a waiting acquisition stops when its context expires.
//
// Given another process holds the lock, when a waiting request reaches its context deadline,
// then acquisition returns an error wrapping the deadline error.
func TestAcquireWaitHonorsContext(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "heartbeat.lock")

	owner, err := Acquire(context.Background(), path, nil)
	if err != nil {
		t.Fatalf("Acquire() owner error = %v", err)
	}

	defer func() {
		if err := owner.Close(); err != nil {
			t.Errorf("owner.Close() error = %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = Acquire(ctx, path, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Acquire() error = %v, want context deadline exceeded", err)
	}
}

// TestReadOwnerPIDReturnsUnavailableSentinel verifies an empty PID record is reported as unavailable.
//
// Given an empty lock file, when its owner PID is read, then the unavailable sentinel -1 is returned.
func TestReadOwnerPIDReturnsUnavailableSentinel(t *testing.T) {
	t.Parallel()

	file, err := os.CreateTemp(t.TempDir(), "heartbeat.lock")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Errorf("file.Close() error = %v", err)
		}
	}()

	if got := readOwnerPID(file); got != -1 {
		t.Fatalf("readOwnerPID() = %d, want -1", got)
	}
}
