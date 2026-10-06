package lock

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const (
	lockFileMode  = 0o600
	retryInterval = 100 * time.Millisecond
)

// Lock represents an acquired exclusive process lock.
type Lock struct {
	file *os.File
}

// Acquire obtains an exclusive lock on path, waiting until it is available or ctx is canceled.
// It logs once if another process holds the lock, using PID -1 when its PID is unavailable.
func Acquire(ctx context.Context, path string) (*Lock, error) {
	file, err := openLockFile(path)
	if err != nil {
		return nil, err
	}

	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	reportedWait := false

	for {
		// Do not acquire the lock if the request was already canceled.
		if err := ctx.Err(); err != nil {
			_ = file.Close()

			return nil, fmt.Errorf("wait for process lock %s: %w", path, err)
		}

		acquired, busy, err := tryAcquire(file)
		if err != nil {
			_ = file.Close()

			return nil, fmt.Errorf("acquire process lock %s: %w", path, err)
		}

		if acquired {
			if err := writeOwnerPID(file); err != nil {
				_ = file.Close()

				return nil, fmt.Errorf("write owner PID to process lock %s: %w", path, err)
			}

			return &Lock{file: file}, nil
		}

		if !busy {
			// Retry immediately after an interrupt; only lock contention needs a delay.
			continue
		}

		if !reportedWait {
			slog.Info("waiting for another process to finish", "pid", readOwnerPID(file))

			reportedWait = true
		}

		select {
		case <-ctx.Done():
			_ = file.Close()

			return nil, fmt.Errorf("wait for process lock %s: %w", path, ctx.Err())
		case <-ticker.C:
		}
	}
}

// openLockFile opens the lock path and verifies it refers to a regular file.
func openLockFile(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, lockFileMode)
	if err != nil {
		return nil, fmt.Errorf("open process lock %s: %w", path, err)
	}

	file := os.NewFile(uintptr(fd), path)

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()

		return nil, fmt.Errorf("stat process lock %s: %w", path, err)
	}

	if !info.Mode().IsRegular() {
		_ = file.Close()

		return nil, fmt.Errorf("process lock %s is not a regular file", path)
	}

	return file, nil
}

// tryAcquire attempts to lock file without blocking and reports whether it is busy.
func tryAcquire(file *os.File) (bool, bool, error) {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err == nil {
		return true, false, nil
	}

	if errors.Is(err, unix.EINTR) {
		return false, false, nil
	}

	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return false, true, nil
	}

	return false, false, fmt.Errorf("flock lock file: %w", err)
}

// Close clears the recorded PID and releases the process lock by closing its underlying file.
func (lock *Lock) Close() error {
	clearErr := clearOwnerPID(lock.file)
	closeErr := lock.file.Close()

	return errors.Join(clearErr, closeErr)
}

// readOwnerPID returns the PID recorded in file, or -1 when the record is missing or invalid.
func readOwnerPID(file *os.File) int {
	var data [32]byte

	n, err := file.ReadAt(data[:], 0)
	if err != nil && !errors.Is(err, io.EOF) {
		return -1
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data[:n])))
	if err != nil || pid <= 0 {
		return -1
	}

	return pid
}

// writeOwnerPID records the current process ID while it holds the lock.
func writeOwnerPID(file *os.File) error {
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("truncate process lock PID: %w", err)
	}

	pid := []byte(strconv.Itoa(os.Getpid()) + "\n")

	n, err := file.WriteAt(pid, 0)
	if err != nil {
		return fmt.Errorf("write process lock PID: %w", err)
	}

	if n != len(pid) {
		return fmt.Errorf("write process lock PID: %w", io.ErrShortWrite)
	}

	return nil
}

// clearOwnerPID removes the recorded process ID before the lock is released.
func clearOwnerPID(file *os.File) error {
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("clear process lock PID: %w", err)
	}

	return nil
}
