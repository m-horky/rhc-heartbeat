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

// TestFilesystemReadRejectsSymlink verifies that Read does not follow symlinks.
//
// Given a symlink to a regular file, when Filesystem.Read reads the symlink,
// then it returns an error.
func TestFilesystemReadRejectsSymlink(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	target := filepath.Join(directory, "target.conf")
	link := filepath.Join(directory, "link.conf")

	if err := os.WriteFile(target, []byte("configuration"), 0o640); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	_, err := (Filesystem{}).Read(link)
	if err == nil {
		t.Fatal("Read() succeeded for a symlink")
	}
}

// TestFilesystemStatRejectsSymlink verifies that Stat rejects symlinks.
//
// Given a symlink to a regular file, when Filesystem.Stat inspects the symlink,
// then it returns an error instead of target metadata.
func TestFilesystemStatRejectsSymlink(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	target := filepath.Join(directory, "target.conf")
	link := filepath.Join(directory, "link.conf")

	if err := os.WriteFile(target, []byte("configuration"), 0o640); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	_, err := (Filesystem{}).Stat(link)
	if err == nil {
		t.Fatal("Stat() succeeded for a symlink")
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
