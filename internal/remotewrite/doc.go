// Package remotewrite encodes and uploads heartbeat samples using Prometheus Remote Write 1.0.
//
// New constructs a client from resolved heartbeat configuration and options.
// Upload sends heartbeat records to the configured endpoint; CloseIdleConnections
// releases idle transport connections.
//
//	cfg, _ := config.Get()
//	client, err := remotewrite.New(cfg, remotewrite.Options{UserAgent: "rhc-heartbeat"})
//	if err != nil {
//		return err
//	}
//	defer client.CloseIdleConnections()
//	return client.Upload(ctx, heartbeats)
package remotewrite
