//go:build linux

package cache

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

// TestLoadMissingFileReturnsEmptyCache verifies that a missing cache file is treated as empty.
//
// Given a path with no cache file, when Load reads it, then it returns an empty cache without an error.
func TestLoadMissingFileReturnsEmptyCache(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := loaded.Read(); len(got) != 0 {
		t.Fatalf("Read() = %#v, want empty", got)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cache file stat error = %v, want not-exist", err)
	}
}

// TestSetKeepsNewestHeartbeatPerBoot verifies that Set keeps the highest monotonic time per boot ID.
//
// Given multiple heartbeats for one boot and one for another,
// when they are set, then only the newest per boot is returned.
func TestSetKeepsNewestHeartbeatPerBoot(t *testing.T) {
	t.Parallel()

	loaded, err := Load(filepath.Join(t.TempDir(), "heartbeat.jsonl"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	first := heartbeat.Heartbeat{BootID: "boot-a", TimeMonotonic: time.Second, HostID: "first"}
	newest := heartbeat.Heartbeat{BootID: "boot-a", TimeMonotonic: 3 * time.Second, HostID: "newest"}
	older := heartbeat.Heartbeat{BootID: "boot-a", TimeMonotonic: 2 * time.Second, HostID: "older"}
	otherBoot := heartbeat.Heartbeat{BootID: "boot-b", TimeMonotonic: time.Second, HostID: "other"}

	loaded.Add(first)
	loaded.Add(newest)
	loaded.Add(older)
	loaded.Add(otherBoot)

	want := []heartbeat.Heartbeat{newest, otherBoot}
	if got := loaded.Read(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Read() = %#v, want %#v", got, want)
	}
}

// TestReadReturnsCopy verifies that modifying a returned slice does not mutate the cache.
//
// Given a cache with one heartbeat, when a caller changes the slice returned by Read,
// then the cached heartbeat remains unchanged.
func TestReadReturnsCopy(t *testing.T) {
	t.Parallel()

	loaded, err := Load(filepath.Join(t.TempDir(), "heartbeat.jsonl"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := heartbeat.Heartbeat{BootID: "boot-a", TimeMonotonic: time.Second}
	loaded.Add(want)

	got := loaded.Read()
	got[0].BootID = "mutated"

	if cached := loaded.Read(); !reflect.DeepEqual(cached, []heartbeat.Heartbeat{want}) {
		t.Fatalf("Read() after mutation = %#v, want %#v", cached, []heartbeat.Heartbeat{want})
	}
}

// TestSavePersistsLatestHeartbeats verifies that Save writes the current in-memory values.
//
// Given a cache with multiple values for one boot,
// when Save persists it and Load reads it again, then only the newest value is present.
func TestSavePersistsLatestHeartbeats(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	loaded.Add(heartbeat.Heartbeat{BootID: "boot-a", TimeMonotonic: time.Second, HostID: "old"})
	want := heartbeat.Heartbeat{BootID: "boot-a", TimeMonotonic: 2 * time.Second, HostID: "new"}
	loaded.Add(want)

	if err := loaded.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() after Save error = %v", err)
	}

	if got := reloaded.Read(); !reflect.DeepEqual(got, []heartbeat.Heartbeat{want}) {
		t.Fatalf("Read() after Save = %#v, want %#v", got, []heartbeat.Heartbeat{want})
	}
}

// TestLoadDeduplicatesPersistedHeartbeats verifies that older JSONL records are collapsed by boot ID.
//
// Given a cache file with repeated boot IDs,
// when Load reads it, then it retains the record with the highest monotonic time.
func TestLoadDeduplicatesPersistedHeartbeats(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")
	contents := "{\"BootID\":\"boot-a\",\"TimeMonotonic\":1000000000,\"HostID\":\"old\"}\n" +
		"{\"BootID\":\"boot-b\",\"TimeMonotonic\":1000000000,\"HostID\":\"other\"}\n" +
		"{\"BootID\":\"boot-a\",\"TimeMonotonic\":2000000000,\"HostID\":\"new\"}\n"

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := []heartbeat.Heartbeat{
		{BootID: "boot-a", TimeMonotonic: 2 * time.Second, HostID: "new"},
		{BootID: "boot-b", TimeMonotonic: time.Second, HostID: "other"},
	}
	if got := loaded.Read(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Read() = %#v, want %#v", got, want)
	}
}

// TestClearAndSavePersistsEmptyCache verifies that clearing memory and saving removes persisted heartbeats.
//
// Given a saved cache, when Clear is followed by Save, then loading the file returns an empty cache.
func TestClearAndSavePersistsEmptyCache(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	loaded.Add(heartbeat.Heartbeat{BootID: "boot-a", TimeMonotonic: time.Second})

	if err := loaded.Save(); err != nil {
		t.Fatalf("Save() initial error = %v", err)
	}

	loaded.Clear()

	if got := loaded.Read(); len(got) != 0 {
		t.Fatalf("Read() after Clear = %#v, want empty", got)
	}

	if err := loaded.Save(); err != nil {
		t.Fatalf("Save() after Clear error = %v", err)
	}

	reloaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() after Clear error = %v", err)
	}

	if got := reloaded.Read(); len(got) != 0 {
		t.Fatalf("Read() after Clear and Save = %#v, want empty", got)
	}
}
