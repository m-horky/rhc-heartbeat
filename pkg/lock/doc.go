// Package lock provides the application's context-aware process lock.
//
// Acquire uses the default rhc-heartbeat lock path. It returns when the lock
// is acquired or the context is canceled. Close releases the lock; the lock
// file itself may remain on disk after release.
//
//	ctx := context.Background()
//	processLock, err := lock.Acquire(ctx)
//	if err != nil {
//		return err
//	}
//	defer processLock.Close()
package lock
