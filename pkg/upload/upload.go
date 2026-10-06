package upload

import (
	"context"
	"errors"
	"fmt"

	internalremotewrite "github.com/m-horky/rhc-heartbeat/internal/remotewrite"
	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/constants"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"github.com/m-horky/rhc-heartbeat/pkg/version"
)

// Client uploads heartbeat batches using the resolved application defaults.
type Client struct {
	implementation *internalremotewrite.Client
}

// New constructs a Remote Write client using configuration and certificate path overrides.
func New(cfg config.Config) (*Client, error) {
	options := internalremotewrite.Options{
		ClientCertificatePath: constants.PathFromEnv(
			constants.ClientCertificatePathEnv,
			constants.DefaultClientCertificatePath,
		),
		ClientKeyPath: constants.PathFromEnv(
			constants.ClientKeyPathEnv,
			constants.DefaultClientKeyPath,
		),
		UserAgent: "rhc-heartbeat/" + version.Version,
	}

	implementation, err := internalremotewrite.New(cfg, options)
	if err != nil {
		return nil, fmt.Errorf("create Remote Write client: %w", err)
	}

	return &Client{implementation: implementation}, nil
}

// Upload sends the supplied heartbeats as one Remote Write request.
func (client *Client) Upload(ctx context.Context, heartbeats []heartbeat.Heartbeat) error {
	if client == nil || client.implementation == nil {
		return errors.New("upload heartbeats: missing client")
	}

	if err := client.implementation.Upload(ctx, heartbeats); err != nil {
		return fmt.Errorf("upload heartbeat batch: %w", err)
	}

	return nil
}

// CloseIdleConnections closes idle connections held by the underlying HTTP transport.
func (client *Client) CloseIdleConnections() {
	if client != nil && client.implementation != nil {
		client.implementation.CloseIdleConnections()
	}
}
