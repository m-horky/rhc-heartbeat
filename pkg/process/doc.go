// Package process coordinates heartbeat delivery and local retry handling.
//
// Applications construct a Processor with resolved configuration and a cache.
// Process tries to upload the new heartbeat first. If that fails, it stores the
// heartbeat for retry. If delivery succeeds, it backfills cached heartbeats in
// batches and retains any batch whose delivery is not confirmed.
//
// A caller chooses the cache location, loads configuration, and processes each
// collected heartbeat:
//
//	cfg, err := config.Get()
//	if err != nil {
//		return err
//	}
//	pending := cache.New(cachePath)
//	processor, err := process.New(cfg, pending)
//	if err != nil {
//		return err
//	}
//	defer processor.CloseIdleConnections()
//	if err := processor.Process(ctx, hb); err != nil {
//		return err
//	}
package process
