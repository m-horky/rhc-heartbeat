package cache

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"

	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

const (
	fileMode    = 0o640
	maxFileSize = 1024 * 1024 // 1 MiB
)

// Cache reads and writes pending heartbeat records at its injected path.
type Cache struct {
	filesystem fs.WriteFS
	path       string
}

// New constructs a cache backed by filesystem at path.
func New(filesystem fs.WriteFS, path string) *Cache {
	return &Cache{filesystem: filesystem, path: path}
}

// Read returns cached heartbeats in file order. A missing file produces an empty slice.
func (cache *Cache) Read() ([]heartbeat.Heartbeat, error) {
	data, err := cache.filesystem.Read(cache.path, maxFileSize)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []heartbeat.Heartbeat{}, nil
		}

		return nil, fmt.Errorf("read heartbeat cache %s: %w", cache.path, err)
	}

	heartbeats := make([]heartbeat.Heartbeat, 0)
	if len(data) == 0 {
		return heartbeats, nil
	}

	lines := bytes.Split(data, []byte{'\n'})
	if data[len(data)-1] == '\n' {
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		var hb heartbeat.Heartbeat
		if err := json.Unmarshal(line, &hb); err != nil {
			return nil, fmt.Errorf("decode heartbeat cache %s: %w", cache.path, err)
		}

		heartbeats = append(heartbeats, hb)
	}

	return heartbeats, nil
}

// Write replaces the cache contents with the supplied heartbeats.
func (cache *Cache) Write(heartbeats []heartbeat.Heartbeat) error {
	ordered := append([]heartbeat.Heartbeat(nil), heartbeats...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].TimeUnix.Before(ordered[j].TimeUnix)
	})

	var data bytes.Buffer

	for _, hb := range ordered {
		line, err := json.Marshal(hb)
		if err != nil {
			return fmt.Errorf("encode heartbeat cache: %w", err)
		}

		if data.Len()+len(line)+1 > maxFileSize {
			return fmt.Errorf("encode heartbeat cache: cache exceeds %d bytes", maxFileSize)
		}

		if _, err := data.Write(line); err != nil {
			return fmt.Errorf("encode heartbeat cache: %w", err)
		}

		if err := data.WriteByte('\n'); err != nil {
			return fmt.Errorf("encode heartbeat cache: %w", err)
		}
	}

	if err := cache.filesystem.Replace(cache.path, data.Bytes(), fileMode); err != nil {
		return fmt.Errorf("replace heartbeat cache %s: %w", cache.path, err)
	}

	slog.Debug("heartbeat cache updated", "count", len(heartbeats), "path", cache.path)

	return nil
}
