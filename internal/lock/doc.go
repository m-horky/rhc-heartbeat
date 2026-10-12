// Package lock provides process-level file locking.
//
// Acquire opens or creates a regular lock file and waits until it can obtain an
// exclusive lock. The lock is released when its Close method is called. A
// canceled context causes Acquire to return an error. If another process holds
// the lock, Acquire calls its onWait callback once with the recorded PID, or -1
// if it cannot be read.
//
//	ctx := context.Background()
//	lock, err := lock.Acquire(ctx, "/run/myapp.lock", nil)
//	if err != nil {
//	    return err
//	}
//	defer lock.Close()
package lock
