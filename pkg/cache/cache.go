package cache

import (
	"fmt"

	internalcache "github.com/m-horky/rhc-heartbeat/internal/cache"
	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

// Cache stores the newest pending heartbeat for each boot ID in memory.
// Callers must serialize access to its methods.
type Cache struct {
	implementation *internalcache.Cache
	heartbeats     []heartbeat.Heartbeat
}

// Load reads pending heartbeats from path. A missing file produces an empty cache.
func Load(path string) (*Cache, error) {
	implementation := internalcache.New(fs.Filesystem{}, path)

	heartbeats, err := implementation.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("load pending heartbeats: %w", err)
	}

	loaded := &Cache{implementation: implementation}
	for _, hb := range heartbeats {
		loaded.Add(hb)
	}

	return loaded, nil
}

// Read returns a copy of all cached heartbeats.
func (cache *Cache) Read() []heartbeat.Heartbeat {
	return append([]heartbeat.Heartbeat{}, cache.heartbeats...)
}

// Add adds a heartbeat or replaces its boot ID's value when its monotonic time is newer.
func (cache *Cache) Add(hb heartbeat.Heartbeat) {
	for index, current := range cache.heartbeats {
		if current.BootID == hb.BootID {
			if hb.TimeMonotonic > current.TimeMonotonic {
				cache.heartbeats[index] = hb
			}

			return
		}
	}

	cache.heartbeats = append(cache.heartbeats, hb)
}

// Clear removes all cached heartbeats from memory. Call Save to persist the empty cache.
func (cache *Cache) Clear() {
	cache.heartbeats = nil
}

// Save persists the current in-memory heartbeats to disk. Add and Clear changes
// are not persisted until Save is called; on failure, the in-memory cache remains
// changed and the on-disk state may or may not have been updated.
func (cache *Cache) Save() error {
	if err := cache.implementation.Rewrite(cache.heartbeats); err != nil {
		return fmt.Errorf("save pending heartbeats: %w", err)
	}

	return nil
}
