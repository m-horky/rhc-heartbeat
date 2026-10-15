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

// TestRotate verifies that Rotate renames the configured cache file aside.
//
// Given a configured cache file that cannot be read, when Rotate rotates it
// and the cache is loaded again, then the broken file is kept at a
// timestamped path and the load returns an empty cache.
func TestRotate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")
	t.Setenv(string(constants.PendingCachePath), path)

	if err := os.WriteFile(path, []byte("not a heartbeat record\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("Load() succeeded for a corrupt cache file")
	}

	rotated, err := Rotate()
	if err != nil {
		t.Fatalf("Rotate() error = %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cache file stat error = %v, want not-exist", err)
	}

	if _, err := os.Stat(rotated); err != nil {
		t.Fatalf("rotated file stat error = %v", err)
	}

	reloaded, err := Load()
	if err != nil {
		t.Fatalf("Load() after Rotate() error = %v", err)
	}

	if got := reloaded.Read(); len(got) != 0 {
		t.Fatalf("Read() after rotation = %#v, want empty", got)
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

	hostID := "host-a"
	first := heartbeat.Heartbeat{BootID: "boot-a", TimeMonotonic: time.Second, HostID: &hostID}
	newest := heartbeat.Heartbeat{BootID: "boot-a", TimeMonotonic: 3 * time.Second, HostID: &hostID}
	older := heartbeat.Heartbeat{BootID: "boot-a", TimeMonotonic: 2 * time.Second, HostID: &hostID}
	otherBoot := heartbeat.Heartbeat{BootID: "boot-b", TimeMonotonic: time.Second, HostID: new("other")}

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

// identityMergeCase describes the identity and ordering inputs for a cache merge test.
type identityMergeCase struct {
	name          string
	firstHostID   *string
	nextHostID    *string
	nextTime      time.Duration
	wantHostID    *string
	wantHeartbeat int
}

// TestDeduplication verifies that missing cached identities can be filled by new heartbeats.
//
// Given two heartbeats with the same boot ID and any pair of missing or present host IDs,
// when both heartbeats are added, then they merge if the cached ID is missing or both IDs match.
func TestDeduplication(t *testing.T) {
	tests := []identityMergeCase{
		{name: "both nil", wantHeartbeat: 1},
		{name: "nil then empty", nextHostID: new(""), wantHeartbeat: 1},
		{name: "empty then nil", firstHostID: new(""), wantHeartbeat: 1},
		{name: "both empty", firstHostID: new(""), nextHostID: new(""), wantHeartbeat: 1},
		{name: "nil then present", nextHostID: new("host-a"), wantHostID: new("host-a"), wantHeartbeat: 1},
		{name: "present then nil", firstHostID: new("host-a"), wantHeartbeat: 2},
		{
			name:          "empty then present",
			firstHostID:   new(""),
			nextHostID:    new("host-a"),
			wantHostID:    new("host-a"),
			wantHeartbeat: 1,
		},
		{name: "present then empty", firstHostID: new("host-a"), nextHostID: new(""), wantHeartbeat: 2},
		{name: "same present ID", firstHostID: new("host-a"), nextHostID: new("host-a"), wantHeartbeat: 1},
		{name: "different present IDs", firstHostID: new("host-a"), nextHostID: new("host-b"), wantHeartbeat: 2},
		{
			name:          "new UUID fills older heartbeat",
			nextHostID:    new("host-a"),
			nextTime:      time.Millisecond,
			wantHostID:    new("host-a"),
			wantHeartbeat: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pending := &Cache{}
			pending.Add(heartbeat.Heartbeat{
				BootID:        "boot-a",
				TimeMonotonic: time.Second,
				HostID:        test.firstHostID,
			})

			if test.nextTime == 0 {
				test.nextTime = 2 * time.Second
			}

			pending.Add(heartbeat.Heartbeat{
				BootID:        "boot-a",
				TimeMonotonic: test.nextTime,
				HostID:        test.nextHostID,
			})

			got := pending.Read()
			if len(got) != test.wantHeartbeat {
				t.Fatalf("Read() has %d heartbeats, want %d", len(got), test.wantHeartbeat)
			}

			if test.wantHostID != nil && (got[0].HostID == nil || *got[0].HostID != *test.wantHostID) {
				t.Fatalf("Read()[0].HostID = %v, want %q", got[0].HostID, *test.wantHostID)
			}
		})
	}
}

// TestBindIdentity verifies identity backfilling preserves known fields.
//
// Given cached heartbeats with missing or existing identity fields, when BindIdentity runs,
// then it fills only missing fields and leaves existing identity untouched.
func TestBindIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "heartbeat.jsonl")
	t.Setenv(string(constants.PendingCachePath), path)

	pending, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	pending.Add(heartbeat.Heartbeat{
		BootID:     "boot-a",
		ProductIDs: new([]string{"product-a"}),
	},
	)
	pending.Add(heartbeat.Heartbeat{
		BootID:        "boot-b",
		TimeMonotonic: time.Second,
		HostID:        new("original-host"),
		HostOrg:       new("original-org"),
	})
	pending.Add(heartbeat.Heartbeat{
		BootID:        "boot-b",
		TimeMonotonic: 2 * time.Second,
	},
	)
	pending.BindIdentity(consumer.Identity{UUID: "host-a", OrgID: "org-a"})

	want := []heartbeat.Heartbeat{
		{BootID: "boot-a", HostID: new("host-a"), HostOrg: new("org-a"), ProductIDs: new([]string{"product-a"})},
		{BootID: "boot-b", TimeMonotonic: time.Second, HostID: new("original-host"), HostOrg: new("original-org")},
		{BootID: "boot-b", TimeMonotonic: 2 * time.Second, HostID: new("host-a"), HostOrg: new("org-a")},
	}
	if got := pending.Read(); !cmp.Equal(got, want) {
		t.Fatalf("Read() after BindIdentity() = %#v, want %#v", got, want)
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
