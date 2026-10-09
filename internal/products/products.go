//go:build linux

package products

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"

	"github.com/m-horky/rhc-heartbeat/internal/fs"
)

// ListIDs returns sorted, unique product IDs found in PEM certificates under directories.
// Missing directories are ignored; other filesystem and certificate errors are returned.
func ListIDs(filesystem fs.FS, directories ...string) ([]string, error) {
	ids := make(map[string]struct{})

	for _, directory := range directories {
		entries, err := filesystem.ReadDir(directory)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf("list product certificates in %s: %w", directory, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".pem" {
				continue
			}

			path := filepath.Join(directory, entry.Name())

			data, err := filesystem.Read(path)
			if err != nil {
				return nil, fmt.Errorf("read product certificate %s: %w", path, err)
			}

			certificate, err := parseCertificate(data)
			if err != nil {
				return nil, fmt.Errorf("parse product certificate %s: %w", path, err)
			}

			for _, id := range productIDs(certificate) {
				ids[id] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}

	sort.Strings(result)

	return result, nil
}

// parseCertificate decodes one PEM-encoded X.509 certificate.
func parseCertificate(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("expected PEM CERTIFICATE block")
	}

	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("invalid X.509 certificate: %w", err)
	}

	return certificate, nil
}

// productIDs extracts the final OID arc from each Red Hat product extension.
func productIDs(certificate *x509.Certificate) []string {
	productExtensionPrefix := [...]int{1, 3, 6, 1, 4, 1, 2312, 9, 1}

	var ids []string

	for _, extension := range certificate.Extensions {
		id := extension.Id
		if len(id) != len(productExtensionPrefix)+1 {
			continue
		}

		if slices.Equal(id[:len(productExtensionPrefix)], productExtensionPrefix[:]) {
			ids = append(ids, strconv.Itoa(id[len(id)-1]))
		}
	}

	return ids
}
