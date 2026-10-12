// Package upload configures heartbeat delivery to a Prometheus Remote Write endpoint.
//
// Use New to build a HTTP client. Upload sends a batch; CloseIdleConnections releases
// idle connections when the client is no longer needed.
//
//	cfg, _ := config.Get()
//	client, _ := upload.New(cfg)
//	defer client.CloseIdleConnections()
//	return client.Upload(ctx, heartbeats)
package upload
