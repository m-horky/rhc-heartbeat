// Package cache provides filesystem-backed storage for pending heartbeats.
//
// Load reads the cache from the system's configured path. A missing cache file
// produces an empty cache. Add and Clear change the in-memory cache; call Save to
// persist those changes. Cache methods are not thread-safe, so callers must
// serialize access. For each boot ID, the cache keeps the heartbeat with the
// greatest monotonic time.
//
//	pending, err := cache.Load()
//	if err != nil {
//	    return err
//	}
//	pending.Add(hb)
//	if err := pending.Save(); err != nil {
//	    return err
//	}
package cache
