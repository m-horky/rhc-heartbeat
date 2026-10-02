// Package consumer exposes consumer identity loaded from the system certificate.
package consumer

import (
	"fmt"
	"log/slog"
	"os"

	internalconsumer "github.com/m-horky/rhc-heartbeat/internal/consumer"
	"github.com/m-horky/rhc-heartbeat/internal/fs"
)

const (
	certificatePathEnv     = "RHC_HEARTBEAT_CONSUMER_CERT"
	defaultCertificatePath = "/etc/pki/consumer/cert.pem"
)

// Identity contains the UUID and organization ID from a consumer certificate.
type Identity = internalconsumer.Identity

// Get reads the consumer identity using the configured certificate path or system default.
func Get() (Identity, error) {
	path := os.Getenv(certificatePathEnv)
	if path == "" {
		path = defaultCertificatePath
		slog.Debug("using default consumer certificate", "path", path)
	} else {
		slog.Debug("using configured consumer certificate", "path", path)
	}

	identity, err := internalconsumer.ReadCertificate(fs.Filesystem{}, path)
	if err != nil {
		return Identity{}, fmt.Errorf("load consumer identity: %w", err)
	}

	return identity, nil
}
