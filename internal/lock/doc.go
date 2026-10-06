// Package lock provides context-aware exclusive process-level file locking.
//
// Acquire opens or creates a regular lock file and waits until it can obtain an
// exclusive lock. The lock is released when its Close method is called. A
// canceled context causes Acquire to return an error. If another process holds
// the lock, Acquire logs its recorded PID once, or -1 if it cannot be read.
//
// Usage:
//
//	ctx := context.Background()
//	lock, err := lock.Acquire(ctx, "/run/myapp.lock")
//	if err != nil {
//	    return err
//	}
//	defer lock.Close()
//
// The lock file may remain after Close; its presence does not mean the lock is
// held. The operating system releases the lock when the owning file is closed
// or the process exits. Callers should use the same path for processes that
// coordinate with one another.
package lock
