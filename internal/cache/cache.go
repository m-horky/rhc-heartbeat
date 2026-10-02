// Package cache stores pending heartbeats as JSONL on the local filesystem.
package cache

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

const (
	fileMode    = 0o640
	maxLineSize = 1024
	maxFileSize = 1024 * 1024
)

// Cache reads and writes pending heartbeat records at its injected path.
type Cache struct {
	filesystem fs.WriteFS
	path       string
	mu         sync.Mutex
}

// New constructs a cache backed by filesystem at path.
func New(filesystem fs.WriteFS, path string) *Cache {
	return &Cache{filesystem: filesystem, path: path}
}

// ReadAll returns the cached heartbeats in file order. A missing cache is empty.
func (cache *Cache) ReadAll() ([]heartbeat.Heartbeat, error) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	data, err := cache.readFile()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []heartbeat.Heartbeat{}, nil
		}

		return nil, fmt.Errorf("read heartbeat cache %s: %w", cache.path, err)
	}

	heartbeats := make([]heartbeat.Heartbeat, 0)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, maxLineSize), maxLineSize)

	for scanner.Scan() {
		if len(scanner.Bytes()) > maxLineSize {
			return nil, fmt.Errorf("decode heartbeat cache %s: line exceeds %d bytes", cache.path, maxLineSize)
		}

		var hb heartbeat.Heartbeat
		if err := json.Unmarshal(scanner.Bytes(), &hb); err != nil {
			return nil, fmt.Errorf("decode heartbeat cache %s: %w", cache.path, err)
		}

		heartbeats = append(heartbeats, hb)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan heartbeat cache %s: %w", cache.path, err)
	}

	return heartbeats, nil
}

// Append adds one heartbeat as a JSON value followed by a newline.
func (cache *Cache) Append(hb heartbeat.Heartbeat) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	data, err := json.Marshal(hb)
	if err != nil {
		return fmt.Errorf("encode heartbeat for cache: %w", err)
	}

	if len(data) > maxLineSize {
		return fmt.Errorf("encode heartbeat for cache: line exceeds %d bytes", maxLineSize)
	}

	data = append(data, '\n')

	current, err := cache.readFile()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read heartbeat cache before append %s: %w", cache.path, err)
	}

	if len(current)+len(data) > maxFileSize {
		return fmt.Errorf("append heartbeat cache %s: cache would exceed %d bytes", cache.path, maxFileSize)
	}

	if err := cache.filesystem.Append(cache.path, data, fileMode); err != nil {
		return fmt.Errorf("append heartbeat cache %s: %w", cache.path, err)
	}

	return nil
}

// Rewrite atomically replaces the cache with the supplied heartbeat records.
func (cache *Cache) Rewrite(heartbeats []heartbeat.Heartbeat) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	var data bytes.Buffer

	for _, hb := range heartbeats {
		line, err := json.Marshal(hb)
		if err != nil {
			return fmt.Errorf("encode heartbeat cache: %w", err)
		}

		if len(line) > maxLineSize {
			return fmt.Errorf("encode heartbeat cache: line exceeds %d bytes", maxLineSize)
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

	return nil
}

// readFile opens the cache and returns its contents up to the configured file-size limit.
func (cache *Cache) readFile() ([]byte, error) {
	file, err := cache.filesystem.Open(cache.path)
	if err != nil {
		return nil, fmt.Errorf("open cache file %s: %w", cache.path, err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()

		return nil, fmt.Errorf("stat cache file: %w", err)
	}

	if !info.IsRegular {
		_ = file.Close()

		return nil, fmt.Errorf("cache path %s is not a regular file", cache.path)
	}

	data, readErr := io.ReadAll(io.LimitReader(file, maxFileSize+1))
	closeErr := file.Close()

	if readErr != nil {
		return nil, fmt.Errorf("read cache file: %w", readErr)
	}

	if closeErr != nil {
		return nil, fmt.Errorf("close cache file: %w", closeErr)
	}

	if len(data) > maxFileSize {
		return nil, fmt.Errorf("cache file exceeds %d bytes", maxFileSize)
	}

	return data, nil
}
