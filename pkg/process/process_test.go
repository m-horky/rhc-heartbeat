package process

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/m-horky/rhc-heartbeat/pkg/cache"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

// TestProcessCachesNewHeartbeatAfterUploadFailure verifies that an undelivered new heartbeat is persisted.
//
// Given an uploader that fails, when Process handles a heartbeat, then it is appended
// to the cache and Process returns no error if caching succeeds.
func TestProcessCachesNewHeartbeatAfterUploadFailure(t *testing.T) {
	t.Parallel()

	pending := cache.New(filepath.Join(t.TempDir(), "heartbeat.jsonl"))
	want := heartbeat.Heartbeat{HostID: "new"}
	calls := 0
	upload := func(context.Context, []heartbeat.Heartbeat) error {
		calls++

		return errors.New("unavailable")
	}

	if err := Process(context.Background(), want, upload, pending); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if calls != 1 {
		t.Fatalf("upload calls = %d, want 1", calls)
	}

	got, err := pending.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if !reflect.DeepEqual(got, []heartbeat.Heartbeat{want}) {
		t.Fatalf("ReadAll() = %#v, want [%#v]", got, want)
	}
}

// TestProcessRetainsCacheWhenBackfillFails verifies that an unsuccessful backfill does not remove pending records.
//
// Given cached heartbeats and an uploader that accepts the new heartbeat but fails
// the backfill, when Process runs, then the cached heartbeats remain unchanged.
func TestProcessRetainsCacheWhenBackfillFails(t *testing.T) {
	t.Parallel()

	pending := cache.New(filepath.Join(t.TempDir(), "heartbeat.jsonl"))

	cached := []heartbeat.Heartbeat{{HostID: "old-1"}, {HostID: "old-2"}}
	if err := pending.Rewrite(cached); err != nil {
		t.Fatalf("Rewrite() error = %v", err)
	}

	calls := 0
	upload := func(_ context.Context, batch []heartbeat.Heartbeat) error {
		calls++
		if calls == 1 {
			if len(batch) != 1 || batch[0].HostID != "new" {
				t.Fatalf("first upload batch = %#v, want new heartbeat", batch)
			}

			return nil
		}

		return errors.New("unavailable")
	}

	if err := Process(context.Background(), heartbeat.Heartbeat{HostID: "new"}, upload, pending); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if calls != 2 {
		t.Fatalf("upload calls = %d, want 2", calls)
	}

	got, err := pending.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if !reflect.DeepEqual(got, cached) {
		t.Fatalf("ReadAll() = %#v, want %#v", got, cached)
	}
}

// TestProcessRemovesCachedRecordsAfterConfirmedBackfill verifies that successful delivery removes cached records.
//
// Given cached heartbeats and an uploader that confirms every delivery, when Process
// runs, then the cache is empty after its batches have been delivered.
func TestProcessRemovesCachedRecordsAfterConfirmedBackfill(t *testing.T) {
	t.Parallel()

	pending := cache.New(filepath.Join(t.TempDir(), "heartbeat.jsonl"))

	cached := []heartbeat.Heartbeat{{HostID: "old"}}
	if err := pending.Rewrite(cached); err != nil {
		t.Fatalf("Rewrite() error = %v", err)
	}

	calls := 0

	upload := func(context.Context, []heartbeat.Heartbeat) error {
		calls++

		return nil
	}
	if err := Process(context.Background(), heartbeat.Heartbeat{HostID: "new"}, upload, pending); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	if calls != 2 {
		t.Fatalf("upload calls = %d, want 2", calls)
	}

	got, err := pending.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("ReadAll() = %#v, want empty cache", got)
	}
}

// TestProcessSerializesConcurrentCalls verifies that concurrent processing calls do not interleave.
//
// Given two concurrent calls using one cache, when the first upload is blocked, then the second upload waits.
func TestProcessSerializesConcurrentCalls(t *testing.T) {
	t.Parallel()

	pending := cache.New(filepath.Join(t.TempDir(), "heartbeat.jsonl"))
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	upload := func(context.Context, []heartbeat.Heartbeat) error {
		started <- struct{}{}

		<-release

		return nil
	}
	results := make(chan error, 2)

	go func() {
		results <- Process(context.Background(), heartbeat.Heartbeat{HostID: "first"}, upload, pending)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first upload did not start")
	}

	go func() {
		results <- Process(context.Background(), heartbeat.Heartbeat{HostID: "second"}, upload, pending)
	}()

	select {
	case <-started:
		close(release)
		<-results
		<-results
		t.Fatal("second upload started before the first call completed")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)

	for range 2 {
		if err := <-results; err != nil {
			t.Fatalf("Process() error = %v", err)
		}
	}
}

// TestProcessBackfillsCachedHeartbeatsInBatches verifies that backfill uploads are bounded.
//
// Given more cached heartbeats than one upload batch, when Process backfills them,
// then it sends separate bounded batches and empties the cache after success.
func TestProcessBackfillsCachedHeartbeatsInBatches(t *testing.T) {
	t.Parallel()

	pending := cache.New(filepath.Join(t.TempDir(), "heartbeat.jsonl"))

	cached := make([]heartbeat.Heartbeat, uploadBatchSize+1)
	for i := range cached {
		cached[i].HostID = "pending"
	}

	if err := pending.Rewrite(cached); err != nil {
		t.Fatalf("Rewrite() error = %v", err)
	}

	var batchSizes []int

	upload := func(_ context.Context, batch []heartbeat.Heartbeat) error {
		batchSizes = append(batchSizes, len(batch))

		return nil
	}
	if err := Process(context.Background(), heartbeat.Heartbeat{HostID: "new"}, upload, pending); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	wantBatchSizes := []int{1, uploadBatchSize, 1}
	if !reflect.DeepEqual(batchSizes, wantBatchSizes) {
		t.Fatalf("upload batch sizes = %v, want %v", batchSizes, wantBatchSizes)
	}

	got, err := pending.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("ReadAll() returned %d records, want empty cache", len(got))
	}
}
