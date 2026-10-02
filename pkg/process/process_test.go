package process

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/m-horky/rhc-heartbeat/pkg/cache"
	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"go.opentelemetry.io/collector/pdata/plog"
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

	if err := processHeartbeat(context.Background(), want, upload, pending); err != nil {
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

	if err := processHeartbeat(context.Background(), heartbeat.Heartbeat{HostID: "new"}, upload, pending); err != nil {
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
	if err := processHeartbeat(context.Background(), heartbeat.Heartbeat{HostID: "new"}, upload, pending); err != nil {
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
		results <- processHeartbeat(context.Background(), heartbeat.Heartbeat{HostID: "first"}, upload, pending)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first upload did not start")
	}

	go func() {
		results <- processHeartbeat(context.Background(), heartbeat.Heartbeat{HostID: "second"}, upload, pending)
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
	if err := processHeartbeat(context.Background(), heartbeat.Heartbeat{HostID: "new"}, upload, pending); err != nil {
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

// TestNewProcessorUploadsThroughConfiguredOTLP verifies the processor wires its OTLP client to the upload workflow.
//
// Given a configured OTLP endpoint and one cached heartbeat
// When the processor handles a new heartbeat
// Then it uploads the new heartbeat separately, backfills the cache, and clears delivered records.
func TestNewProcessorUploadsThroughConfiguredOTLP(t *testing.T) {
	t.Parallel()

	batchSizes := make(chan int, 2)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		payload, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}

		logs, err := (&plog.JSONUnmarshaler{}).UnmarshalLogs(payload)
		if err != nil {
			t.Errorf("decode OTLP/JSON request: %v", err)
		} else {
			batchSizes <- logs.ResourceLogs().Len()
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"partialSuccess":{}}`))
	}))
	defer server.Close()

	pending := cache.New(filepath.Join(t.TempDir(), "heartbeat.jsonl"))
	if err := pending.Append(heartbeat.Heartbeat{HostID: "cached"}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	processor, err := New(config.Config{
		OTEL: config.Endpoint{URI: server.URL + "/v1/logs", TLSVerify: true},
	}, pending)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer processor.CloseIdleConnections()

	if err := processor.Process(context.Background(), heartbeat.Heartbeat{HostID: "new"}); err != nil {
		t.Fatalf("Process() error = %v", err)
	}

	for range 2 {
		if got := <-batchSizes; got != 1 {
			t.Errorf("OTLP request contained %d records, want one", got)
		}
	}

	got, err := pending.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("ReadAll() returned %d records, want empty cache", len(got))
	}
}

// processHeartbeat creates a processor with a test upload callback and runs one heartbeat through it.
func processHeartbeat(
	ctx context.Context,
	hb heartbeat.Heartbeat,
	upload func(context.Context, []heartbeat.Heartbeat) error,
	pending *cache.Cache,
) error {
	return (&Processor{upload: upload, pending: pending}).Process(ctx, hb)
}
