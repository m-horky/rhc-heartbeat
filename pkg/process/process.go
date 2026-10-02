package process

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/m-horky/rhc-heartbeat/pkg/cache"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

const uploadBatchSize = 100

// Process serializes calls that share the pending cache instance, uploads a new
// heartbeat before attempting to backfill cached records, and leaves uncertain
// deliveries pending for a later attempt. Upload errors for the new heartbeat
// are handled by caching it; errors while backfilling retain pending records.
func Process(
	ctx context.Context,
	hb heartbeat.Heartbeat,
	upload func(context.Context, []heartbeat.Heartbeat) error,
	pending *cache.Cache,
) error {
	if upload == nil {
		return errors.New("upload heartbeat: missing upload function")
	}

	if pending == nil {
		return errors.New("process heartbeat: missing cache")
	}

	if err := pending.WithExclusive(func() error {
		return process(ctx, hb, upload, pending)
	}); err != nil {
		return fmt.Errorf("process heartbeat: %w", err)
	}

	return nil
}

// process uploads a heartbeat and updates its pending cache while its caller holds exclusivity.
func process(
	ctx context.Context,
	hb heartbeat.Heartbeat,
	upload func(context.Context, []heartbeat.Heartbeat) error,
	pending *cache.Cache,
) error {
	if err := upload(ctx, []heartbeat.Heartbeat{hb}); err != nil {
		slog.Warn("new heartbeat upload failed; caching for retry", "err", err)

		if cacheErr := pending.Append(hb); cacheErr != nil {
			return errors.Join(fmt.Errorf("upload new heartbeat: %w", err), cacheErr)
		}

		return nil
	}

	cached, err := pending.ReadAll()
	if err != nil {
		return fmt.Errorf("read pending heartbeats: %w", err)
	}

	for len(cached) > 0 {
		batchSize := min(uploadBatchSize, len(cached))
		if err := upload(ctx, cached[:batchSize]); err != nil {
			slog.Warn("cached heartbeat backfill failed; retaining pending records", "err", err)

			return nil
		}

		cached = cached[batchSize:]
		if err := pending.Rewrite(cached); err != nil {
			return fmt.Errorf("remove delivered heartbeats from cache: %w", err)
		}
	}

	return nil
}
