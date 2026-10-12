package consumer

import (
	"fmt"

	"github.com/m-horky/rhc-heartbeat/internal/fs"
	internalconsumer "github.com/m-horky/rhc-heartbeat/internal/rhsmx509/consumer"
	"github.com/m-horky/rhc-heartbeat/pkg/constants"
)

// Identity contains the UUID and organization ID from a consumer certificate.
type Identity = internalconsumer.Identity

// Get reads the consumer identity using the configured certificate path or system default.
func Get() (Identity, error) {
	path := constants.Path(constants.IdentityCertPath)

	identity, err := internalconsumer.Read(fs.Filesystem{}, path)
	if err != nil {
		return Identity{}, fmt.Errorf("load consumer identity: %w", err)
	}

	return identity, nil
}
