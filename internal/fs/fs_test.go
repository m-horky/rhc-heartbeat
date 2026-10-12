package fs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestFilesystemRead verifies that a regular file's contents are returned.
//
// Given a readable regular file within the byte limit, when Filesystem.Read
// reads its path, then it returns the complete file contents without error.
func TestFilesystemRead(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	path := filepath.Join(directory, "configuration.conf")

	contents := []byte("configuration")
	if err := os.WriteFile(path, contents, 0o640); err != nil {
		t.Fatal(err)
	}

	got, err := (Filesystem{}).Read(path, 1024)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if string(got) != string(contents) {
		t.Fatalf("Read() = %q, want %q", got, contents)
	}
}

// TestErrNotExist verifies that missing files preserve the
// os.ErrNotExist condition.
//
// Given a path that does not exist, when Filesystem.Read reads it, then the
// returned error matches os.ErrNotExist.
func TestErrNotExist(t *testing.T) {
	t.Parallel()

	_, err := (Filesystem{}).Read(filepath.Join(t.TempDir(), "missing.conf"), 1024)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Read() error = %v, want os.ErrNotExist", err)
	}
}

// TestErrTruncated verifies that oversized file contents are bounded.
//
// Given a regular file larger than the byte limit, when Filesystem.Read reads
// it, then it returns only the limit and reports ErrTruncated.
func TestErrTruncated(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "large.conf")
	if err := os.WriteFile(path, []byte("12345"), 0o640); err != nil {
		t.Fatal(err)
	}

	got, err := (Filesystem{}).Read(path, 3)
	if !errors.Is(err, ErrTruncated) {
		t.Fatalf("Read() error = %v, want ErrTruncated", err)
	}

	if string(got) != "123" {
		t.Fatalf("Read() = %q, want %q", got, "123")
	}
}

// TestReplace verifies that Replace writes replacement contents.
//
// Given a file with old contents, when Filesystem.Replace writes new contents,
// then subsequent reads return the complete new contents and mode.
func TestReplace(t *testing.T) {
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
