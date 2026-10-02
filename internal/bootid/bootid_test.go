package bootid

import (
	"errors"
	"io/fs"
	"testing"

	internalfs "github.com/m-horky/rhc-heartbeat/internal/fs"
)

// TestReadReturnsTrimmedBootID verifies file contents are trimmed without UUID validation.
//
// Given boot ID file contents with surrounding whitespace, when reading the ID,
// then the trimmed value is returned as-is.
func TestReadReturnsTrimmedBootID(t *testing.T) {
	t.Parallel()

	filesystem := &testFilesystem{data: []byte("  kernel-value\n\t")}

	got, err := Read(filesystem)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if got != "kernel-value" {
		t.Errorf("Read() = %q, want %q", got, "kernel-value")
	}

	if filesystem.path != path {
		t.Errorf("Read() path = %q, want %q", filesystem.path, path)
	}
}

// TestReadWrapsFilesystemErrors verifies boot ID read failures remain identifiable.
//
// Given a filesystem read error, when reading the boot ID, then the original error is wrapped and returned.
func TestReadWrapsFilesystemErrors(t *testing.T) {
	t.Parallel()

	wantErr := fs.ErrPermission
	filesystem := &testFilesystem{err: wantErr}

	_, err := Read(filesystem)
	if !errors.Is(err, wantErr) {
		t.Fatalf("Read() error = %v, want wrapped %v", err, wantErr)
	}
}

type testFilesystem struct {
	internalfs.FS

	data []byte
	err  error
	path string
}

// Read records the requested path and returns the configured test file contents or error.
func (filesystem *testFilesystem) Read(path string) ([]byte, error) {
	filesystem.path = path

	return filesystem.data, filesystem.err
}
