package consumer

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestReadCertificateExtractsIdentity verifies identity extraction from the certificate subject.
//
// Given a PEM certificate with a common name and one organization,
// when reading it, then both identity fields are returned.
func TestReadCertificateExtractsIdentity(t *testing.T) {
	path := filepath.Join(t.TempDir(), "consumer.pem")
	writeCertificateTestFile(t, path, makeCertificateTestPEM(t, pkix.Name{
		CommonName:   "c94f4db8-ad9c-4653-95fd-647d247420f8",
		Organization: []string{"20008437"},
	}))

	got, err := ReadCertificate(path)
	if err != nil {
		t.Fatalf("ReadCertificate() error = %v", err)
	}

	if got.UUID != "c94f4db8-ad9c-4653-95fd-647d247420f8" || got.OrgID != "20008437" {
		t.Errorf("ReadCertificate() = %+v, want UUID and organization from certificate subject", got)
	}
}

// TestReadCertificateRejectsInvalidIdentity verifies malformed certificates and incomplete subjects are rejected.
//
// Given a malformed certificate or missing or ambiguous identity fields,
// when reading it, then an error is returned.
func TestReadCertificateRejectsInvalidIdentity(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr string
	}{
		{
			name:    "not PEM",
			data:    []byte("not a certificate"),
			wantErr: "expected PEM CERTIFICATE block",
		},
		{
			name:    "invalid X.509",
			data:    pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("invalid")}),
			wantErr: "invalid X.509 certificate",
		},
		{
			name:    "missing common name",
			data:    makeCertificateTestPEM(t, pkix.Name{Organization: []string{"20008437"}}),
			wantErr: "no common name",
		},
		{
			name:    "missing organization",
			data:    makeCertificateTestPEM(t, pkix.Name{CommonName: "consumer-uuid"}),
			wantErr: "exactly one organization",
		},
		{
			name: "multiple organizations",
			data: makeCertificateTestPEM(t, pkix.Name{
				CommonName:   "consumer-uuid",
				Organization: []string{"20008437", "20008438"},
			}),
			wantErr: "exactly one organization",
		},
		{
			name:    "empty organization",
			data:    makeCertificateTestPEM(t, pkix.Name{CommonName: "consumer-uuid", Organization: []string{"  "}}),
			wantErr: "empty organization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "consumer.pem")
			writeCertificateTestFile(t, path, tt.data)

			if _, err := ReadCertificate(path); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ReadCertificate() error = %v, want text %q", err, tt.wantErr)
			}
		})
	}
}

// makeCertificateTestPEM creates a self-signed PEM certificate with the supplied subject.
func makeCertificateTestPEM(t *testing.T, subject pkix.Name) []byte {
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

// writeCertificateTestFile writes a private certificate fixture.
func writeCertificateTestFile(t *testing.T, path string, data []byte) {
	t.Helper()

	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write certificate fixture: %v", err)
	}
}
