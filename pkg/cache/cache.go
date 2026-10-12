package cache

import (
	"fmt"
	"log/slog"
	"sort"

	internalcache "github.com/m-horky/rhc-heartbeat/internal/cache"
	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/constants"
	"github.com/m-horky/rhc-heartbeat/pkg/consumer"
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
func (cache *Cache) Add(hb heartbeat.Heartbeat) {
	for i, existing := range cache.heartbeats {
		if existing.BootID != hb.BootID {
			continue
		}

		replaced := false

		if hb.TimeMonotonic > existing.TimeMonotonic {
			if hb.HostID == nil || *hb.HostID == "" {
				hb.HostID = existing.HostID
				replaced = true
			}

			if hb.HostOrg == nil || *hb.HostOrg == "" {
				hb.HostOrg = existing.HostOrg
				replaced = true
			}

			cache.heartbeats[i] = hb
			if replaced {
				slog.Debug("updated heartbeat host", "boot-id", hb.BootID)
			}
		}

		sortHeartbeats(cache.heartbeats)

		return
	}

	cache.heartbeats = append(cache.heartbeats, hb)
	sortHeartbeats(cache.heartbeats)
}

// sortHeartbeats orders heartbeats by their wall-clock timestamp.
func sortHeartbeats(heartbeats []heartbeat.Heartbeat) {
	sort.SliceStable(heartbeats, func(i, j int) bool {
		return heartbeats[i].TimeUnix.Before(heartbeats[j].TimeUnix)
	})
}

// BindIdentity fills missing identity fields on cached heartbeats without replacing known values.
func (cache *Cache) BindIdentity(identity consumer.Identity) {
	for i := range cache.heartbeats {
		if cache.heartbeats[i].HostID == nil || *cache.heartbeats[i].HostID == "" {
			cache.heartbeats[i].HostID = new(identity.UUID)
		}

		if cache.heartbeats[i].HostOrg == nil || *cache.heartbeats[i].HostOrg == "" {
			cache.heartbeats[i].HostOrg = new(identity.OrgID)
		}
	}
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
