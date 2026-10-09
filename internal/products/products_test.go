//go:build linux

package products

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	iofs "io/fs"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	internalfs "github.com/m-horky/rhc-heartbeat/internal/fs"
)

// TestListIDsReturnsSortedUniqueProductIDs verifies IDs are extracted from product extension OIDs.
//
// Given product certificates in both directories and a non-product certificate,
// when listing IDs, then matching IDs are deduplicated and returned in sorted order.
func TestListIDsReturnsSortedUniqueProductIDs(t *testing.T) {
	t.Parallel()

	filesystem := newProductTestFS(t, map[string]map[string][]byte{
		"/virtual/product-default": {
			"479.pem": makeProductTestPEM(t, 479),
		},
		"/virtual/product": {
			"479.pem":       makeProductTestPEM(t, 479),
			"2048.pem":      makeProductTestPEM(t, 2048),
			"unrelated.pem": makeProductTestPEM(t),
			"ignored.txt":   []byte("not a certificate"),
		},
	})

	got, err := ListIDs(filesystem, "/virtual/product-default", "/virtual/product")
	if err != nil {
		t.Fatalf("ListIDs() error = %v", err)
	}

	if want := []string{"2048", "479"}; !reflect.DeepEqual(got, want) {
		t.Errorf("ListIDs() = %v, want %v", got, want)
	}
}

// TestProductIDsIgnoresNestedOIDs verifies only exact product extension OIDs are extracted.
//
// Given a certificate with an exact product OID and a descendant OID,
// when product IDs are extracted, then only the exact product OID is returned.
func TestProductIDsIgnoresNestedOIDs(t *testing.T) {
	t.Parallel()

	certificate := &x509.Certificate{
		Extensions: []pkix.Extension{
			{Id: []int{1, 3, 6, 1, 4, 1, 2312, 9, 1, 479}},
			{Id: []int{1, 3, 6, 1, 4, 1, 2312, 9, 1, 479, 3}},
		},
	}

	got := productIDs(certificate)
	if want := []string{"479"}; !reflect.DeepEqual(got, want) {
		t.Errorf("productIDs() = %v, want %v", got, want)
	}
}

// TestListIDsIgnoresMissingDirectories verifies absent certificate directories are optional.
//
// Given a missing certificate directory, when listing IDs, then it is treated as empty.
func TestListIDsIgnoresMissingDirectories(t *testing.T) {
	t.Parallel()

	filesystem := productTestFS{
		directoryErrors: map[string]error{"/virtual/missing": iofs.ErrNotExist},
	}

	got, err := ListIDs(filesystem, "/virtual/missing")
	if err != nil {
		t.Fatalf("ListIDs() error = %v", err)
	}

	if len(got) != 0 {
		t.Errorf("ListIDs() = %v, want no IDs", got)
	}
}

// TestListIDsReturnsDirectoryErrors verifies unexpected directory-listing errors are preserved.
//
// Given a filesystem error while listing a certificate directory,
// when listing IDs, then the error is returned with its cause intact.
func TestListIDsReturnsDirectoryErrors(t *testing.T) {
	t.Parallel()

	wantErr := iofs.ErrPermission
	filesystem := productTestFS{
		directoryErrors: map[string]error{"/virtual/product": wantErr},
	}

	if _, err := ListIDs(filesystem, "/virtual/product"); !errors.Is(err, wantErr) {
		t.Fatalf("ListIDs() error = %v, want wrapped %v", err, wantErr)
	}
}

// TestListIDsRejectsMalformedCertificates verifies unreadable certificate data is not silently ignored.
//
// Given a PEM candidate that does not contain a certificate,
// when listing IDs, then a contextual parse error is returned.
func TestListIDsRejectsMalformedCertificates(t *testing.T) {
	t.Parallel()

	filesystem := newProductTestFS(t, map[string]map[string][]byte{
		"/virtual/product": {"broken.pem": []byte("not a certificate")},
	})

	if _, err := ListIDs(filesystem, "/virtual/product"); err == nil {
		t.Fatal("ListIDs() error = nil, want malformed certificate error")
	}
}

// makeProductTestPEM creates a PEM certificate with the supplied product ID extensions.
func makeProductTestPEM(t *testing.T, ids ...int) []byte {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate test key: %v", err)
	}

	extraExtensions := make([]pkix.Extension, 0, len(ids))

	productOIDPrefix := []int{1, 3, 6, 1, 4, 1, 2312, 9, 1}
	for _, id := range ids {
		oid := append(append([]int(nil), productOIDPrefix...), id)
		extraExtensions = append(extraExtensions, pkix.Extension{Id: oid, Value: []byte{0x05, 0x00}})
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Minute),
		BasicConstraintsValid: true,
		ExtraExtensions:       extraExtensions,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create test certificate: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// newProductTestFS creates a fake filesystem containing the supplied virtual directories and files.
func newProductTestFS(t *testing.T, directories map[string]map[string][]byte) productTestFS {
	t.Helper()

	filesystem := productTestFS{
		directories: make(map[string][]os.DirEntry, len(directories)),
		files:       make(map[string][]byte),
	}
	for virtualDirectory, files := range directories {
		actualDirectory := t.TempDir()
		for name, contents := range files {
			if err := os.WriteFile(filepath.Join(actualDirectory, name), contents, 0o600); err != nil {
				t.Fatalf("write product test file: %v", err)
			}

			filesystem.files[filepath.Join(virtualDirectory, name)] = contents
		}

		entries, err := os.ReadDir(actualDirectory)
		if err != nil {
			t.Fatalf("read product test directory: %v", err)
		}

		filesystem.directories[virtualDirectory] = entries
	}

	return filesystem
}

type productTestFS struct {
	internalfs.FS

	directories     map[string][]os.DirEntry
	files           map[string][]byte
	directoryErrors map[string]error
	readErrors      map[string]error
}

var _ internalfs.FS = productTestFS{}

// ReadDir returns configured directory entries or a virtual not-exist error.
func (filesystem productTestFS) ReadDir(path string) ([]os.DirEntry, error) {
	if err := filesystem.directoryErrors[path]; err != nil {
		return nil, err
	}

	entries, ok := filesystem.directories[path]
	if !ok {
		return nil, &iofs.PathError{Op: "readdir", Path: path, Err: iofs.ErrNotExist}
	}

	return entries, nil
}

// Read returns configured file contents or a virtual not-exist error.
func (filesystem productTestFS) Read(path string) ([]byte, error) {
	if err := filesystem.readErrors[path]; err != nil {
		return nil, err
	}

	contents, ok := filesystem.files[path]
	if !ok {
		return nil, &iofs.PathError{Op: "read", Path: path, Err: iofs.ErrNotExist}
	}

	return append([]byte(nil), contents...), nil
}
