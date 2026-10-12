// Package cache provides filesystem-backed storage for pending heartbeats.
//
// Load reads the cache from the system's configured path. A missing cache file
// produces an empty cache. Add and Clear change the in-memory cache; call Save to
// persist those changes. Cache methods are not thread-safe, so callers must
// serialize access. For each boot ID, the cache keeps the heartbeat with the
// greatest monotonic time.
//
// BindIdentity fills missing identity fields without replacing values already
// recorded on a heartbeat.
//
//	identity, _ := consumer.Get()
//	pending, _ := cache.Load()
//	pending.BindIdentity(identity)
//	hb, _ := heartbeat.Get(heartbeat.KindPing)
//	pending.Add(hb)
//	_ = pending.Save();
package cache
