package remotewrite

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/snappy"
	"github.com/m-horky/rhc-heartbeat/internal/fs"
	"github.com/m-horky/rhc-heartbeat/pkg/config"
	"github.com/m-horky/rhc-heartbeat/pkg/heartbeat"
	"golang.org/x/net/http/httpproxy"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

// mustURL parses a test URL or fails the active test.
func mustURL(t *testing.T, value string) url.URL {
	t.Helper()

	parsed, err := url.Parse(value)
	if err != nil {
		return url.URL{Path: value}
	}

	return *parsed
}

// TestHeadersBody verifies Remote Write v1 request encoding and headers.
//
// Given a valid heartbeat and endpoint, when Upload sends the payload,
// then it sends block-compressed protobuf with the required protocol headers.
func TestHeadersBody(t *testing.T) {
	requests := make(chan http.Header, 1)
	schema := newRemoteWriteMessages()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests <- request.Header.Clone()

		payload, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)

			return
		}

		decoded, err := snappy.Decode(nil, payload)
		if err != nil {
			t.Errorf("decode Snappy block: %v", err)

			return
		}

		message := dynamicpb.NewMessage(schema.writeRequest)
		if err := proto.Unmarshal(decoded, message); err != nil {
			t.Errorf("decode protobuf WriteRequest: %v", err)
		}

		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := New(
		config.Config{Heartbeat: config.Endpoint{URI: mustURL(t, server.URL)}},
		Options{UserAgent: "rhc-heartbeat/test"},
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.CloseIdleConnections()

	hb := heartbeat.Heartbeat{HostID: "uuid", HostOrg: "org", BootID: "boot", Kind: heartbeat.KindOff}
	if err := client.Upload(context.Background(), []heartbeat.Heartbeat{hb}); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	got := <-requests
	if value := got.Get("Content-Encoding"); value != "snappy" {
		t.Errorf("Content-Encoding = %q, want snappy", value)
	}

	if got := got.Get("Content-Type"); got != "application/x-protobuf" {
		t.Errorf("Content-Type = %q, want application/x-protobuf", got)
	}

	if got := got.Get("X-Prometheus-Remote-Write-Version"); got != "0.1.0" {
		t.Errorf("Remote Write version = %q, want 0.1.0", got)
	}

	if got := got.Get("User-Agent"); got != "rhc-heartbeat/test" {
		t.Errorf("User-Agent = %q, want rhc-heartbeat/test", got)
	}
}

// TestProxyBypassRules verifies configured bypass rules select direct connections correctly.
//
// Given a proxy and a list of bypass rules, when a request target matches a rule,
// then the transport bypasses the proxy and otherwise uses it.
func TestProxyBypassRules(t *testing.T) {
	proxyURI := "http://proxy.example:8080"
	tests := []struct {
		name      string
		rules     string
		target    string
		wantProxy bool
	}{
		{name: "exact hostname", rules: "example.com", target: "https://example.com/path"},
		{name: "hostname subdomain", rules: "example.com", target: "https://api.example.com/path"},
		{name: "DNS label boundary", rules: "example.com", target: "https://notexample.com/path", wantProxy: true},
		{name: "leading wildcard alias", rules: "*.example.com", target: "https://api.example.com/path"},
		{name: "CIDR match", rules: "10.0.0.0/8", target: "http://10.12.3.4/path"},
		{name: "IP literal", rules: "2001:db8::1", target: "https://[2001:db8::1]/path"},
		{name: "matching port", rules: "example.com:443", target: "https://example.com/path"},
		{name: "different port", rules: "example.com:8443", target: "https://example.com/path", wantProxy: true},
		{name: "global wildcard", rules: "*", target: "https://anything.example/path"},
		{name: "ignore malformed rule and keep valid rule", rules: "[bad, example.com", target: "https://example.com/path"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request, err := http.NewRequest(http.MethodGet, test.target, nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}

			client, err := newHTTPClient(Options{}, false, true, &httpproxy.Config{
				HTTPProxy:  proxyURI,
				HTTPSProxy: proxyURI,
				NoProxy:    test.rules,
			})
			if err != nil {
				t.Fatalf("newHTTPClient() error = %v", err)
			}
			defer client.CloseIdleConnections()

			selected, err := client.Transport.(*http.Transport).Proxy(request)
			if err != nil {
				t.Fatalf("select proxy: %v", err)
			}

			if gotProxy := selected != nil; gotProxy != test.wantProxy {
				t.Errorf("proxy selected = %t, want %t", gotProxy, test.wantProxy)
			}
		})
	}
}

// TestTLSConfigAppliesVerificationSetting verifies the configured TLS verification flag is honored.
//
// Given a valid client certificate and CA, when TLS configuration is created,
// then verification follows the configured setting.
func TestTLSConfigAppliesVerificationSetting(t *testing.T) {
	certificate, privateKey := makeCertificatePair(t)
	directory := t.TempDir()
	certificatePath := filepath.Join(directory, "client.pem")
	keyPath := filepath.Join(directory, "client-key.pem")
	caPath := filepath.Join(directory, "ca.pem")

	if err := os.WriteFile(certificatePath, certificate, 0o600); err != nil {
		t.Fatalf("write client certificate: %v", err)
	}

	if err := os.WriteFile(keyPath, privateKey, 0o600); err != nil {
		t.Fatalf("write client key: %v", err)
	}

	if err := os.WriteFile(caPath, certificate, 0o600); err != nil {
		t.Fatalf("write CA certificate: %v", err)
	}

	tests := []struct {
		name string
		in   bool
		want bool
	}{
		{name: "verify", in: true, want: false},
		{name: "skip verification", in: false, want: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := newTLSConfig(Options{
				Filesystem:         fs.Filesystem{},
				CertificatePath:    certificatePath,
				CertificateKeyPath: keyPath,
				CACertificatePath:  caPath,
			}, test.in)
			if err != nil {
				t.Fatalf("newTLSConfig() error = %v", err)
			}

			if got.InsecureSkipVerify != test.want {
				t.Errorf("InsecureSkipVerify = %t, want %t", got.InsecureSkipVerify, test.want)
			}

			if len(got.Certificates) != 1 {
				t.Errorf("client certificates = %d, want 1", len(got.Certificates))
			}
		})
	}
}

// makeCertificatePair creates a temporary self-signed certificate and private key for TLS tests.
func makeCertificatePair(t *testing.T) ([]byte, []byte) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate private key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "rhc-heartbeat test"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})

	encodedKey, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("encode private key: %v", err)
	}

	return certificate, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encodedKey})
}
