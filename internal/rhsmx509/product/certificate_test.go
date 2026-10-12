package product

import (
	"crypto/x509/pkix"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TestReadProductExtensions verifies exact product extension OIDs are extracted.
//
// Given a certificate with product tags, when the product is extracted,
// it contains the expected product IDs.
func TestReadProductExtensions(t *testing.T) {
	t.Parallel()

	extensions := []pkix.Extension{
		{Id: []int{1, 3, 6, 1, 4, 1, 2312, 9, 1, 10, 1}, Value: []byte("RHEL")},
		{Id: []int{1, 3, 6, 1, 4, 1, 2312, 9, 1, 10, 2}, Value: []byte("10.0")},
		{Id: []int{1, 3, 6, 1, 4, 1, 2312, 9, 1, 10, 3}, Value: []byte("x86_64")},
		{Id: []int{1, 3, 6, 1, 4, 1, 2312, 9, 1, 10, 4}, Value: []byte("rhel-10,rhel-10-x84_64")},
		{Id: []int{1, 3, 6, 1, 4, 1, 2312, 9, 1, 9, 1}, Value: []byte("RHEL")},
		{Id: []int{1, 3, 6, 1, 4, 1, 2312, 9, 1, 9, 2}, Value: []byte("9.8,9.10")},
		{Id: []int{1, 3, 6, 1, 4, 1, 2312, 9, 1, 9, 3}, Value: []byte("ppc64le")},
		{Id: []int{1, 3, 6, 1, 4, 1, 2312, 9, 1, 9, 4}, Value: []byte("rhel-9,rhel-9-ppc64le")},
	}

	got, err := readProductExtensions(extensions)
	if err != nil {
		t.Fatal(err)
	}

	want := []Product{
		{
			ID:            9,
			Name:          "RHEL",
			Versions:      []string{"9.8", "9.10"},
			Architectures: []string{"ppc64le"},
			Tags:          []string{"rhel-9", "rhel-9-ppc64le"},
		},
		{
			ID:            10,
			Name:          "RHEL",
			Versions:      []string{"10.0"},
			Architectures: []string{"x86_64"},
			Tags:          []string{"rhel-10", "rhel-10-x84_64"},
		},
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("productIDs() mismatch (-got +want):\n%s", diff)
	}
}
