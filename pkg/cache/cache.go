package cache

import (
	"fmt"
	"log/slog"

	internalcache "github.com/m-horky/rhc-heartbeat/internal/cache"
	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/constants"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

// Cache stores the newest pending heartbeat for each boot ID in memory.
// Callers must serialize access to its methods.
type Cache struct {
	implementation *internalcache.Cache
	heartbeats     []heartbeat.Heartbeat
	persistedCount int
}

// Load reads pending heartbeats from path. A missing file produces an empty cache.
func Load() (*Cache, error) {
	impl := internalcache.New(fs.Filesystem{}, constants.Path(constants.PendingCachePath))

	heartbeats, err := impl.Read()
	if err != nil {
		return nil, fmt.Errorf("load pending heartbeats: %w", err)
	}

	loaded := &Cache{implementation: impl}
	for _, hb := range heartbeats {
		loaded.Add(hb)
	}

	loaded.persistedCount = len(loaded.heartbeats)

	return loaded, nil
}

// Read returns cached heartbeats.
func (cache *Cache) Read() []heartbeat.Heartbeat {
	// append copies heartbeats into a new slice, so callers cannot modify the cache.
	return append([]heartbeat.Heartbeat{}, cache.heartbeats...)
}

// Add adds a heartbeat or replaces its boot ID's value when its monotonic time is newer.
func (cache *Cache) Add(heartbeat heartbeat.Heartbeat) {
	for i, hb := range cache.heartbeats {
		if hb.BootID == heartbeat.BootID {
			if heartbeat.TimeMonotonic > hb.TimeMonotonic {
				cache.heartbeats[i] = heartbeat
			}

			return
		}
	}

	cache.heartbeats = append(cache.heartbeats, heartbeat)
}

// Clear removes all cached heartbeats from memory. Call Save to persist the empty cache.
func (cache *Cache) Clear() {
	cache.heartbeats = nil
}

// Save persists the current in-memory heartbeats to disk. Add and Clear changes
// are not persisted until Save is called.
func (cache *Cache) Save() error {
	if err := cache.implementation.Write(cache.heartbeats); err != nil {
		return fmt.Errorf("save pending heartbeats: %w", err)
	}

	currentCount := len(cache.heartbeats)
	if cache.persistedCount > 0 && currentCount == 0 {
		slog.Debug("heartbeat cache cleared")
	}

	if cache.persistedCount < currentCount {
		slog.Debug("pending heartbeats saved")
	}

	cache.persistedCount = currentCount

	return nil
}
