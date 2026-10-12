package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/m-horky/rhc-heartbeat/pkg/constants"
	"github.com/m-horky/rhc-heartbeat/pkg/consumer"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

// TestMissingIsEmpty verifies that a missing cache file is treated as empty.
//
// Given a path with no cache file, when Load reads it,
// then it returns an empty cache without an error.
func TestMissingIsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")
	t.Setenv(string(constants.PendingCachePath), path)

	loaded, err := Load()
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

// TestLatestWithinBoot verifies that Add keeps the highest monotonic time per boot ID.
//
// Given multiple heartbeats for one boot and one for another,
// when they are set, then only the newest per boot is returned.
func TestLatestWithinBoot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")
	t.Setenv(string(constants.PendingCachePath), path)

	loaded, err := Load()
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
	got := loaded.Read()

	if !cmp.Equal(got, want) {
		t.Fatalf("Read() = %#v, want %#v", got, want)
	}
}

// TestReadReturnsCopy verifies that modifying a returned slice does not mutate the cache.
//
// Given a cache with one heartbeat, when a caller changes the slice returned by Read,
// then the cached heartbeat remains unchanged.
func TestReadReturnsCopy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")
	t.Setenv(string(constants.PendingCachePath), path)

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := heartbeat.Heartbeat{BootID: "boot-a", TimeMonotonic: time.Second}
	loaded.Add(want)

	got := loaded.Read()
	got[0].BootID = "mutated"

	if cached := loaded.Read(); !cmp.Equal(cached, []heartbeat.Heartbeat{want}) {
		t.Fatalf("Read() after mutation = %#v, want %#v", cached, []heartbeat.Heartbeat{want})
	}
}

// TestClear verifies that clearing memory removes persisted heartbeats.
//
// Given a non-empty saved cache, when Clear is followed by Save,
// then loading the file returns an empty cache.
func TestClear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")
	t.Setenv(string(constants.PendingCachePath), path)

	loaded, err := Load()
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

	reloaded, err := Load()
	if err != nil {
		t.Fatalf("Load() after Clear error = %v", err)
	}

	if got := reloaded.Read(); len(got) != 0 {
		t.Fatalf("Read() after Clear and Save = %#v, want empty", got)
	}
}
