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
	"testing"
	"time"

	"github.com/m-horky/rhc-heartbeat/pkg/constants"
)

// TestGetUsesConfiguredCertificatePath verifies the certificate path environment override.
//
// Given a certificate path in the environment, when Get reads identity,
// then it returns the identity from that certificate.
func TestGetUsesConfiguredCertificatePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "consumer.pem")
	writeConsumerTestCertificate(t, path)
	t.Setenv(constants.ClientCertificatePathEnv, path)

	got, err := Get()
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.UUID != "fake-system-uuid" || got.OrgID != "fake-system-org" {
		t.Errorf("Get() = %+v, want identity from configured certificate", got)
	}
}

// writeConsumerTestCertificate writes a PEM certificate fixture with a representative subject.
func writeConsumerTestCertificate(t *testing.T, path string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "fake-system-uuid",
			Organization: []string{"fake-system-org"},
		},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Minute),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create test certificate: %v", err)
	}

	data := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write certificate fixture: %v", err)
	}
}
