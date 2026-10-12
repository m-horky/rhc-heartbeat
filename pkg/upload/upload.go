package upload

import (
	"context"
	"errors"
	"fmt"

	"github.com/m-horky/rhc-heartbeat/internal/fs"
	internalremotewrite "github.com/m-horky/rhc-heartbeat/internal/remotewrite"
	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/constants"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"github.com/m-horky/rhc-heartbeat/pkg/version"
	"golang.org/x/net/http/httpproxy"
)

// Client uploads heartbeat batches using the resolved application defaults.
type Client struct {
	impl *internalremotewrite.Client
}

// New constructs a Remote Write client using configuration and certificate path overrides.
func New(cfg config.Config) (*Client, error) {
	proxyURI := cfg.HTTP.Proxy.URI.String()
	proxyConfig := &httpproxy.Config{
		HTTPProxy:  proxyURI,
		HTTPSProxy: proxyURI,
		NoProxy:    cfg.HTTP.Proxy.NoProxy,
	}

	options := internalremotewrite.Options{
		Filesystem:         fs.Filesystem{},
		CACertificatePath:  cfg.Heartbeat.CAPath,
		CertificatePath:    constants.Path(constants.IdentityCertPath),
		CertificateKeyPath: constants.Path(constants.IdentityKeyPath),
		UserAgent:          "rhc-heartbeat/" + version.Version,
		Proxy:              proxyConfig,
	}

	implementation, err := internalremotewrite.New(cfg, options)
	if err != nil {
		return nil, fmt.Errorf("create Remote Write client: %w", err)
	}

	return &Client{impl: implementation}, nil
}

// Upload sends the supplied heartbeats as one Remote Write request.
func (client *Client) Upload(ctx context.Context, heartbeats []heartbeat.Heartbeat) error {
	if client == nil || client.impl == nil {
		return errors.New("upload heartbeats: missing client")
	}

	if err := client.impl.Upload(ctx, heartbeats); err != nil {
		return fmt.Errorf("upload heartbeat batch: %w", err)
	}

	return nil
}

// CloseIdleConnections closes idle connections held by the underlying HTTP transport.
func (client *Client) CloseIdleConnections() {
	if client != nil && client.impl != nil {
		client.impl.CloseIdleConnections()
	}
}
