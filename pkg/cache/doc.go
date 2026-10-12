// Package cache provides filesystem-backed storage for pending heartbeats.
//
// Load reads the cache file into memory. If the file does not exist, it returns
// an empty cache. Add and Clear are in-memory operations: call Save to persist them.
// Cache methods are not thread-safe.
//
// The cache only keeps the latest heartbeat for each boot ID.
//
//	hb := heartbeat.Heartbeat{}
//	pending, err := cache.Load("/var/lib/rhc/heartbeat.jsonl")
//	if err != nil {
//	    return err
//	}
//	pending.Add(hb)
//	if err := pending.Save(); err != nil {
//	    return err
//	}
package cache
