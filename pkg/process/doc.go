// Package process coordinates heartbeat delivery and local retry handling.
//
// Applications call Process after collecting a heartbeat, supplying the upload
// operation and a cache configured for their environment. Process tries to upload
// the new heartbeat first. If that fails, it stores the heartbeat for retry. If
// delivery succeeds, it attempts to backfill cached heartbeats in batches and
// retains any batch whose delivery is not confirmed.
//
// A caller chooses the cache location, constructs the cache, and processes each
// collected heartbeat:
//
//	pending := cache.New(cachePath)
//	err := process.Process(ctx, hb, uploader.Upload, pending)
//	if err != nil {
//		return err
//	}
package process
