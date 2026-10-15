// Package cache reads and writes pending heartbeats as JSONL.
//
// New creates a cache using a fs.WriteFS implementation and a cache file path.
// Read returns heartbeats in file order, and Write rewrites the cache file in place.
// Rotate renames an unreadable cache file aside, returning the rotated path.
//
//	store := cache.New(fs.Filesystem{}, "/var/lib/rhc/heartbeat.jsonl")
//	heartbeats, err := store.Read()
//	if err != nil {
//		rotated, err := store.Rotate("old")
//		if err != nil {
//			return err
//		}
//		slog.Error("heartbeat cache rotated", "path", rotated)
//		heartbeats = []heartbeat.Heartbeat{}
//	}
//	heartbeats = append(heartbeats, heartbeat.Heartbeat{BootID: "boot-id"})
//	if err := store.Write(heartbeats); err != nil {
//		return err
//	}
package cache
