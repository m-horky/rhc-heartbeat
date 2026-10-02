// Package cache provides filesystem-backed storage for pending heartbeats.
//
// Applications choose the cache file path and construct a cache with New. The
// cache stores each typed heartbeat as one JSON value per line, creates the file
// with mode 0640, and expects its parent directory to be provisioned already.
// Records are limited to 1 KiB each and the cache file is limited to 1 MiB.
// WithExclusive serializes multi-step workflows that use the same Cache instance;
// direct cache operations and other processes do not participate in that lock.
//
//	pending := cache.New("/var/lib/rhc/heartbeat.jsonl")
//	if err := pending.Append(hb); err != nil {
//		return err
//	}
package cache
