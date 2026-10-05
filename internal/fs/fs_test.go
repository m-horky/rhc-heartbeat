//go:build linux

package fs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestFilesystemRead verifies that a regular file's contents are returned.
//
// Given a readable regular file, when Filesystem.Read reads its path, then it
// returns the complete file contents without error.
func TestFilesystemRead(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "configuration.conf")

	contents := []byte("configuration")
	if err := os.WriteFile(path, contents, 0o640); err != nil {
		t.Fatal(err)
	}

	got, err := (Filesystem{}).Read(path)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if string(got) != string(contents) {
		t.Fatalf("Read() = %q, want %q", got, contents)
	}
}

// TestFilesystemReadRejectsDirectory verifies that directories cannot be read
// as configuration files.
//
// Given a directory path, when Filesystem.Read reads it, then it returns an
// error.
func TestFilesystemReadRejectsDirectory(t *testing.T) {
	t.Parallel()

	_, err := (Filesystem{}).Read(t.TempDir())
	if err == nil {
		t.Fatal("Read() succeeded for a directory")
	}
}

// TestFilesystemReadMissingFile verifies that missing files preserve the
// os.ErrNotExist condition.
//
// Given a path that does not exist, when Filesystem.Read reads it, then the
// returned error matches os.ErrNotExist.
func TestFilesystemReadMissingFile(t *testing.T) {
	t.Parallel()

	_, err := (Filesystem{}).Read(filepath.Join(t.TempDir(), "missing.conf"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Read() error = %v, want os.ErrNotExist", err)
	}
}

// TestFilesystemAppendCreatesAndAppends verifies that Append creates a file and adds data.
//
// Given a missing file in an existing directory, when Filesystem.Append appends
// multiple values, then the file contains all values in order.
func TestFilesystemAppendCreatesAndAppends(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "cache.jsonl")

	filesystem := Filesystem{}
	for _, data := range [][]byte{[]byte("first\n"), []byte("second\n")} {
		if err := filesystem.Append(path, data, 0o640); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(got) != "first\nsecond\n" {
		t.Fatalf("file contents = %q, want %q", got, "first\nsecond\n")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	if gotMode := info.Mode().Perm(); gotMode != 0o640 {
		t.Fatalf("file mode = %#o, want %#o", gotMode, 0o640)
	}
}

// TestFilesystemReplaceAtomicallyReplaces verifies that Replace writes replacement contents.
//
// Given a file with old contents, when Filesystem.Replace writes new contents,
// then subsequent reads return the complete new contents and requested mode.
func TestFilesystemReplaceAtomicallyReplaces(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "cache.jsonl")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := (Filesystem{}).Replace(path, []byte("new\n"), 0o640); err != nil {
		t.Fatalf("Replace() error = %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(got) != "new\n" {
		t.Fatalf("file contents = %q, want %q", got, "new\n")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}

	if gotMode := info.Mode().Perm(); gotMode != 0o640 {
		t.Fatalf("file mode = %#o, want %#o", gotMode, 0o640)
	}
}
