// Package cache stores pending heartbeats as JSONL on the local filesystem.
//
// New creates a cache using an fs.WriteFS implementation and a cache file path.
// Append adds one heartbeat, ReadAll loads pending heartbeats, and Rewrite
// atomically replaces the cached records.
//
//	store := cache.New(fs.Filesystem{}, "/var/lib/rhc/heartbeat.jsonl")
//	if err := store.Append(heartbeat.Heartbeat{HostID: "system-uuid"}); err != nil {
//		return err
//	}
//	pending, err := store.ReadAll()
//	if err != nil {
//		return err
//	}
//	_ = pending
package cache
