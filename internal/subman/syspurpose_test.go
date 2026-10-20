package subman

import (
	iofs "io/fs"
	"testing"

	"github.com/google/go-cmp/cmp"
	internalfs "github.com/m-horky/rhc-heartbeat/internal/fs"
)

// TestReadSyspurpose verifies system purpose attributes are extracted.
//
// Given a syspurpose file with role, service level, and usage set, when it is
// read, then each attribute is returned trimmed and other fields are ignored.
func TestReadSyspurpose(t *testing.T) {
	t.Parallel()

	path := "/virtual/syspurpose.json"
	filesystem := virtFS{files: map[string][]byte{
		path: []byte(`{
			"role": "Red Hat Enterprise Linux Server",
			"service_level_agreement": " Premium ",
			"usage": "Production",
			"addons": ["eap"]
		}`),
	}}

	got, err := ReadSyspurpose(filesystem, path)
	if err != nil {
		t.Fatalf("ReadSyspurpose() error = %v", err)
	}

	want := Syspurpose{
		Role:         new("Red Hat Enterprise Linux Server"),
		ServiceLevel: new("Premium"),
		Usage:        new("Production"),
	}
	if !cmp.Equal(got, want) {
		t.Errorf("ReadSyspurpose() = %+v, want %+v", got, want)
	}
}

// TestReadSyspurposeUnset verifies unset attributes normalize to nil.
//
// Given a syspurpose file without attributes or with blank values, when it is
// read, then those attributes are nil rather than empty strings.
func TestReadSyspurposeUnset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		contents string
	}{
		{name: "empty object", contents: "{}"},
		{name: "null", contents: "null"},
		{
			name:     "blank values",
			contents: `{"role": "", "service_level_agreement": "  ", "usage": ""}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := "/virtual/syspurpose.json"
			filesystem := virtFS{files: map[string][]byte{path: []byte(test.contents)}}

			got, err := ReadSyspurpose(filesystem, path)
			if err != nil {
				t.Fatalf("ReadSyspurpose() error = %v", err)
			}

			if !cmp.Equal(got, Syspurpose{}) {
				t.Errorf("ReadSyspurpose() = %+v, want an empty system purpose", got)
			}
		})
	}
}

// TestReadSyspurposeMissingFile verifies a missing file is not an error.
//
// Given a filesystem without a syspurpose file, when it is read, then an
// empty system purpose is returned without an error.
func TestReadSyspurposeMissingFile(t *testing.T) {
	t.Parallel()

	filesystem := virtFS{files: map[string][]byte{}}

	got, err := ReadSyspurpose(filesystem, "/virtual/syspurpose.json")
	if err != nil {
		t.Fatalf("ReadSyspurpose() error = %v", err)
	}

	if !cmp.Equal(got, Syspurpose{}) {
		t.Errorf("ReadSyspurpose() = %+v, want an empty system purpose", got)
	}
}

// TestReadSyspurposeInvalid verifies broken files are reported as errors.
//
// Given a malformed or oversized syspurpose file, when it is read, then an
// explanatory error is returned.
func TestReadSyspurposeInvalid(t *testing.T) {
	t.Parallel()

	oversized := make([]byte, maxSyspurposeSize+1)
	for i := range oversized {
		oversized[i] = 'x'
	}

	tests := []struct {
		name     string
		contents []byte
	}{
		{name: "not JSON", contents: []byte("system purpose")},
		{name: "incomplete JSON", contents: []byte(`{"role": `)},
		{name: "wrong value type", contents: []byte(`{"role": ["server"]}`)},
		{name: "oversized", contents: oversized},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := "/virtual/syspurpose.json"
			filesystem := virtFS{files: map[string][]byte{path: test.contents}}

			if _, err := ReadSyspurpose(filesystem, path); err == nil {
				t.Errorf("ReadSyspurpose() error = nil for %q, want an error", test.name)
			}
		})
	}
}

type virtFS struct {
	internalfs.FS

	files map[string][]byte
}

var _ internalfs.FS = virtFS{}

// Read returns configured test contents up to maxBytes or a virtual not-exist error.
func (filesystem virtFS) Read(path string, maxBytes int64) ([]byte, error) {
	data, ok := filesystem.files[path]
	if !ok {
		return nil, &iofs.PathError{Op: "read", Path: path, Err: iofs.ErrNotExist}
	}

	if int64(len(data)) > maxBytes {
		return append([]byte(nil), data[:maxBytes]...), internalfs.ErrTruncated
	}

	return append([]byte(nil), data...), nil
}
