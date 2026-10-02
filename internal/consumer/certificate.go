// Package consumer reads the identity from a consumer certificate.
package consumer

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Identity contains the UUID and organization ID from a consumer certificate.
type Identity struct {
	UUID  string
	OrgID string
}

// ReadCertificate reads a PEM-encoded consumer certificate and extracts its identity.
func ReadCertificate(path string) (Identity, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Identity{}, fmt.Errorf("read consumer certificate %s: %w", path, err)
	}

	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return Identity{}, fmt.Errorf("parse consumer certificate %s: expected PEM CERTIFICATE block", path)
	}

	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return Identity{}, fmt.Errorf("parse consumer certificate %s: invalid X.509 certificate", path)
	}

	uuid := strings.TrimSpace(certificate.Subject.CommonName)
	if uuid == "" {
		return Identity{}, fmt.Errorf("consumer certificate %s subject has no common name", path)
	}

	if len(certificate.Subject.Organization) != 1 {
		return Identity{}, fmt.Errorf("consumer certificate %s subject must have exactly one organization", path)
	}

	orgID := strings.TrimSpace(certificate.Subject.Organization[0])
	if orgID == "" {
		return Identity{}, fmt.Errorf("consumer certificate %s subject has an empty organization", path)
	}

	return Identity{UUID: uuid, OrgID: orgID}, nil
}
