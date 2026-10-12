package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

	original := []heartbeat.Heartbeat{{HostID: new("old")}, {HostID: new("pending")}}
	if err := cache.Write(original); err != nil {
		t.Fatalf("Write() initial error = %v", err)
	}

	want := []heartbeat.Heartbeat{{HostID: new("pending")}}
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

// TestWriteIsChronological verifies persisted JSONL records are chronological.
//
// Given heartbeats supplied out of timestamp order, when Write persists them,
// then Read returns the records in ascending timestamp order.
func TestWriteIsChronological(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")
	cache := New(fs.Filesystem{}, path)
	later := heartbeat.Heartbeat{BootID: "later", TimeUnix: time.Unix(20, 0)}
	earlier := heartbeat.Heartbeat{BootID: "earlier", TimeUnix: time.Unix(10, 0)}

	if err := cache.Write([]heartbeat.Heartbeat{later, earlier}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	got, err := cache.Read()
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	want := []heartbeat.Heartbeat{earlier, later}
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
