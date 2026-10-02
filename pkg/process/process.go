package process

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/m-horky/rhc-heartbeat/internal/otlp"
	"github.com/m-horky/rhc-heartbeat/pkg/cache"
	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
)

const uploadBatchSize = 100

// Processor uploads newly collected heartbeats and backfills the pending cache.
type Processor struct {
	upload               func(context.Context, []heartbeat.Heartbeat) error
	closeIdleConnections func()
	pending              *cache.Cache
}

// New constructs a heartbeat processor using the resolved application configuration.
func New(cfg config.Config, pending *cache.Cache) (*Processor, error) {
	if pending == nil {
		return nil, errors.New("create heartbeat processor: missing cache")
	}

	client, err := otlp.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("create OTLP client: %w", err)
	}

	return &Processor{
		upload:               client.Upload,
		closeIdleConnections: client.CloseIdleConnections,
		pending:              pending,
	}, nil
}

// Process uploads a new heartbeat before attempting to backfill cached records.
// It caches the new heartbeat when delivery fails and retains backfill batches
// whose delivery is not confirmed.
func (processor *Processor) Process(ctx context.Context, hb heartbeat.Heartbeat) error {
	if processor == nil {
		return errors.New("process heartbeat: missing processor")
	}

	if processor.upload == nil {
		return errors.New("process heartbeat: missing upload client")
	}

	if processor.pending == nil {
		return errors.New("process heartbeat: missing cache")
	}

	if err := processor.pending.WithExclusive(func() error {
		return process(ctx, hb, processor.upload, processor.pending)
	}); err != nil {
		return fmt.Errorf("process heartbeat: %w", err)
	}

	return nil
}

// CloseIdleConnections closes the processor's idle OTLP HTTP connections.
func (processor *Processor) CloseIdleConnections() {
	if processor != nil && processor.closeIdleConnections != nil {
		processor.closeIdleConnections()
	}
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
