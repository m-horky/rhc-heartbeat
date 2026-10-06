// Package upload configures heartbeat delivery to a Prometheus Remote Write endpoint.
//
// New builds a client from the resolved heartbeat configuration. It applies the client certificate and
// key paths from their environment overrides, falling back to the system defaults, and sets the
// application version in the request user agent. Upload sends a batch; CloseIdleConnections releases
// idle connections when the client is no longer needed.
//
//	cfg, err := config.Get()
//	if err != nil {
//		return err
//	}
//	client, err := upload.New(cfg)
//	if err != nil {
//		return err
//	}
//	defer client.CloseIdleConnections()
//	return client.Upload(ctx, heartbeats)
package upload
