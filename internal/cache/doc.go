// Package cache reads and writes pending heartbeats as JSONL.
//
// New creates a cache using a fs.WriteFS implementation and a cache file path.
// Read returns heartbeats in file order, and Write atomically replaces all records.
//
//	store := cache.New(fs.Filesystem{}, "/var/lib/rhc/heartbeat.jsonl")
//	heartbeats, err := store.Read()
//	if err != nil {
//		return err
//	}
//	heartbeats = append(heartbeats, heartbeat.Heartbeat{BootID: "boot-id"})
//	if err := store.Write(heartbeats); err != nil {
//		return err
//	}
package cache
