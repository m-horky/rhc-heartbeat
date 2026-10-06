// Package cache provides filesystem-backed storage for pending heartbeats.
//
// Load reads the cache file into memory. If the file does not exist, Load returns
// an empty cache. The cache keeps only the heartbeat with the highest monotonic
// time for each boot ID. Add and Clear modify memory only; call Save to persist
// those changes. A Save error leaves the changed in-memory state intact and
// does not establish whether the on-disk cache was updated.
//
// The cache stores each heartbeat as one JSON value per line, creates the file
// with mode 0640, and expects its parent directory to be provisioned already.
// Records are limited to 1 KiB each and the cache file is limited to 1 MiB.
// Cache methods are not synchronized; callers must serialize access.
//
//	pending, err := cache.Load("/var/lib/rhc/heartbeat.jsonl")
//	if err != nil {
//	    return err
//	}
//	pending.Add(hb)
//	if err := pending.Save(); err != nil {
//	    return err
//	}
package cache
