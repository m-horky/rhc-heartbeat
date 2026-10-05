//go:build linux

package cache

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

// TestCacheAppendAndReadAll verifies that appended heartbeats round-trip through JSONL.
//
// Given an empty cache, when heartbeats are appended, then ReadAll returns them in order.
func TestCacheAppendAndReadAll(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")
	cache := New(fs.Filesystem{}, path)
	want := []heartbeat.Heartbeat{
		{HostID: "first", TimeUnix: time.Date(2025, 1, 2, 3, 4, 5, 6, time.UTC)},
		{HostID: "second", Kind: heartbeat.KindPing},
	}

	for _, hb := range want {
		if err := cache.Append(hb); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	got, err := cache.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadAll() = %#v, want %#v", got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	if gotMode := info.Mode().Perm(); gotMode != fileMode {
		t.Fatalf("cache file mode = %#o, want %#o", gotMode, fileMode)
	}
}

// TestCacheRewrite verifies that replacement retains only the supplied heartbeats.
//
// Given a cache containing multiple heartbeats, when Rewrite replaces its contents,
// then ReadAll returns exactly the replacement records.
func TestCacheRewrite(t *testing.T) {
	t.Parallel()

	cache := New(fs.Filesystem{}, filepath.Join(t.TempDir(), "heartbeat.jsonl"))

	original := []heartbeat.Heartbeat{{HostID: "old"}, {HostID: "pending"}}
	if err := cache.Rewrite(original); err != nil {
		t.Fatalf("Rewrite() initial error = %v", err)
	}

	want := []heartbeat.Heartbeat{{HostID: "pending"}}
	if err := cache.Rewrite(want); err != nil {
		t.Fatalf("Rewrite() error = %v", err)
	}

	got, err := cache.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReadAll() = %#v, want %#v", got, want)
	}
}

// TestCacheReadRejectsOversizedFile verifies that cache input has a strict total size bound.
//
// Given a cache file larger than 1 MiB, when ReadAll reads it, then it returns an error without reading it all.
func TestCacheReadRejectsOversizedFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")
	if err := os.WriteFile(path, make([]byte, maxFileSize+1), 0o600); err != nil {
		t.Fatal(err)
	}

	cache := New(fs.Filesystem{}, path)
	if _, err := cache.ReadAll(); err == nil {
		t.Fatal("ReadAll() succeeded for an oversized cache file")
	}
}

// TestCacheAppendRejectsOversizedFile verifies that appending cannot exceed the total cache limit.
//
// Given a full cache file, when Append adds another heartbeat, then it returns an error.
func TestCacheAppendRejectsOversizedFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")
	if err := os.WriteFile(path, make([]byte, maxFileSize), 0o600); err != nil {
		t.Fatal(err)
	}

	cache := New(fs.Filesystem{}, path)
	if err := cache.Append(heartbeat.Heartbeat{HostID: "pending"}); err == nil {
		t.Fatal("Append() succeeded when the cache file was full")
	}
}

// TestCacheAppendRejectsOversizedLine verifies that writes obey the JSONL line limit.
//
// Given a heartbeat that encodes to more than 1 KiB, when Append writes it, then it returns an error.
func TestCacheAppendRejectsOversizedLine(t *testing.T) {
	t.Parallel()

	cache := New(fs.Filesystem{}, filepath.Join(t.TempDir(), "heartbeat.jsonl"))
	if err := cache.Append(heartbeat.Heartbeat{HostID: strings.Repeat("x", maxLineSize)}); err == nil {
		t.Fatal("Append() succeeded for an oversized line")
	}
}

// TestCacheReadRejectsOversizedLine verifies that individual JSONL records are bounded.
//
// Given a cache containing a line larger than 1 KiB, when ReadAll reads it, then it returns an error.
func TestCacheReadRejectsOversizedLine(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", maxLineSize+1)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cache := New(fs.Filesystem{}, path)
	if _, err := cache.ReadAll(); err == nil {
		t.Fatal("ReadAll() succeeded for an oversized line")
	}
}

// TestCacheDoesNotCreateDirectory verifies that the provisioned cache directory is required.
//
// Given a missing parent directory, when Append writes a heartbeat, then it fails
// without creating that directory.
func TestCacheDoesNotCreateDirectory(t *testing.T) {
	t.Parallel()

	directory := filepath.Join(t.TempDir(), "not-provisioned")

	cache := New(fs.Filesystem{}, filepath.Join(directory, "heartbeat.jsonl"))
	if err := cache.Append(heartbeat.Heartbeat{HostID: "pending"}); err == nil {
		t.Fatal("Append() succeeded when the cache directory did not exist")
	}

	if _, err := os.Stat(directory); !os.IsNotExist(err) {
		t.Fatalf("cache directory stat error = %v, want not-exist", err)
	}
}
