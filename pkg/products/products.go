//go:build linux

package products

import (
	"fmt"

	"github.com/m-horky/rhc-heartbeat/internal/fs"
	internalproducts "github.com/m-horky/rhc-heartbeat/internal/products"
	"github.com/m-horky/rhc-heartbeat/pkg/constants"
)

// Get returns sorted, unique IDs from the system's Red Hat product certificates.
func Get() ([]string, error) {
	ids, err := internalproducts.ListIDs(
		fs.Filesystem{},
		constants.DefaultProductCertificateDefaultDir,
		constants.DefaultProductCertificateDir,
	)
	if err != nil {
		return nil, fmt.Errorf("load product IDs: %w", err)
	}

	return ids, nil
}
