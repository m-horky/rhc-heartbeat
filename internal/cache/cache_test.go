package cache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

// TestWrite verifies that replacement retains only the supplied heartbeats.
//
// Given a cache containing multiple heartbeats, when Write replaces its contents,
// then Read returns exactly the replacement records.
func TestWrite(t *testing.T) {
	t.Parallel()

	cache := New(fs.Filesystem{}, filepath.Join(t.TempDir(), "heartbeat.jsonl"))

	original := []heartbeat.Heartbeat{{HostID: "old"}, {HostID: "pending"}}
	if err := cache.Write(original); err != nil {
		t.Fatalf("Write() initial error = %v", err)
	}

	want := []heartbeat.Heartbeat{{HostID: "pending"}}
	if err := cache.Write(want); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got, err := cache.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if !cmp.Equal(got, want) {
		t.Fatalf("Read() = %#v, want %#v", got, want)
	}
}

// TestMaxFileSize verifies that cache input has a strict total size bound.
//
// Given a cache file larger than 1 MiB, when Read reads it, then it returns an error without reading it all.
func TestMaxFileSize(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")
	if err := os.WriteFile(path, make([]byte, maxFileSize+1), 0o600); err != nil {
		t.Fatal(err)
	}

	cache := New(fs.Filesystem{}, path)
	if _, err := cache.Read(); err == nil {
		t.Fatal("Read() succeeded for an oversized cache file")
	}
}

// TestMaxLineSize verifies that individual JSONL records are bounded.
//
// Given a cache containing a line larger than configured amount, when Read reads it,
// then it returns an error.
func TestMaxLineSize(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", maxLineSize+1)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cache := New(fs.Filesystem{}, path)
	if _, err := cache.Read(); err == nil {
		t.Fatal("Read() succeeded for an oversized line")
	}
}
