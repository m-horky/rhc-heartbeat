package cache

import (
	"fmt"
	"sync"

	"github.com/m-horky/rhc-heartbeat/internal/cache"
	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

// Cache stores pending heartbeats at an application-supplied path.
type Cache struct {
	implementation *cache.Cache
	mu             sync.Mutex
}

// New constructs a filesystem-backed cache at path. The parent directory must
// already exist and is never created by the cache.
func New(path string) *Cache {
	return &Cache{implementation: cache.New(fs.Filesystem{}, path)}
}

// WithExclusive runs operation while excluding other WithExclusive operations on this cache.
// It supports multi-step workflows that must not interleave, such as upload and backfill.
func (cache *Cache) WithExclusive(operation func() error) error {
	if operation == nil {
		return fmt.Errorf("run cache operation: missing operation")
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	return operation()
}

// ReadAll returns cached heartbeats in file order. A missing cache is empty.
func (cache *Cache) ReadAll() ([]heartbeat.Heartbeat, error) {
	heartbeats, err := cache.implementation.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read pending heartbeats: %w", err)
	}

	return heartbeats, nil
}

// Append adds one heartbeat to the JSONL cache.
func (cache *Cache) Append(hb heartbeat.Heartbeat) error {
	if err := cache.implementation.Append(hb); err != nil {
		return fmt.Errorf("append pending heartbeat: %w", err)
	}

	return nil
}

// Rewrite atomically replaces the cache with the supplied heartbeats.
func (cache *Cache) Rewrite(heartbeats []heartbeat.Heartbeat) error {
	if err := cache.implementation.Rewrite(heartbeats); err != nil {
		return fmt.Errorf("rewrite pending heartbeats: %w", err)
	}

	return nil
}
