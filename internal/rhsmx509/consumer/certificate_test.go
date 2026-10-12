package consumer

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	iofs "io/fs"
	"math/big"
	"testing"
	"time"

	internalfs "github.com/m-horky/rhc-heartbeat/internal/fs"
)

// TestRead verifies identity extraction from the certificate subject.
//
// Given a PEM certificate with a common name and one organization,
// when reading it, then both identity fields are returned.
func TestRead(t *testing.T) {
	path := "/virtual/consumer.pem"
	contents := newCertificate(t, pkix.Name{
		CommonName:   "fake-uuid",
		Organization: []string{"fake-org"},
	})
	filesystem := virtFS{files: map[string][]byte{path: contents}}

	got, err := Read(filesystem, path)
	if err != nil {
		t.Fatalf("ReadCertificate() error = %v", err)
	}

	if got.UUID != "fake-uuid" || got.OrgID != "fake-org" {
		t.Errorf("ReadCertificate() = %+v, want UUID and organization from certificate subject", got)
	}
}

// newCertificate creates a self-signed PEM certificate with the supplied subject.
func newCertificate(t *testing.T, subject pkix.Name) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               subject,
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Minute),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create test certificate: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

type virtFS struct {
	internalfs.FS

	files map[string][]byte
}

var _ internalfs.FS = virtFS{}

// Read returns configured test contents or a virtual not-exist error.
func (filesystem virtFS) Read(path string) ([]byte, error) {
	data, ok := filesystem.files[path]
	if !ok {
		return nil, &iofs.PathError{Op: "read", Path: path, Err: iofs.ErrNotExist}
	}

	return append([]byte(nil), data...), nil
}
