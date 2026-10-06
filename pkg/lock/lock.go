package lock

import (
	"context"
	"fmt"
	"log/slog"

	internallock "github.com/m-horky/rhc-heartbeat/internal/lock"
	"github.com/m-horky/rhc-heartbeat/pkg/constants"
)

// Lock represents an acquired exclusive application process lock.
type Lock struct {
	implementation *internallock.Lock
}

// Acquire obtains the application process lock, waiting until it is available or ctx is canceled.
func Acquire(ctx context.Context) (*Lock, error) {
	implementation, err := internallock.Acquire(ctx, constants.DefaultProcessLockPath, func(pid int) {
		slog.Info("waiting for another process to finish", "pid", pid)
	})
	if err != nil {
		return nil, fmt.Errorf("acquire application process lock: %w", err)
	}

	return &Lock{implementation: implementation}, nil
}

// Close releases the acquired application process lock.
func (lock *Lock) Close() error {
	if lock == nil || lock.implementation == nil {
		return nil
	}

	if err := lock.implementation.Close(); err != nil {
		return fmt.Errorf("release application lock: %w", err)
	}

	return nil
}
