package fs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// FileInfo contains the filesystem information about a file.
// Its shape follows the information exposed by os.FileInfo
// while keeping ownership inside this package.
type FileInfo struct {
	Mode      os.FileMode
	UID       uint32
	GID       uint32
	IsDir     bool
	IsRegular bool
}

// File is the read-only subset of *os.File used by consumers of FS.
type File interface {
	io.Reader
	Stat() (FileInfo, error)
	Close() error
}

// ErrTruncated indicates that a read returned only the first maxBytes bytes.
var ErrTruncated = errors.New("file read truncated")

// FS is the filesystem contract used by loaders. Read returns ErrTruncated
// when the file contains more than the requested number of bytes.
type FS interface {
	Read(path string, maxBytes int64) ([]byte, error)
	ReadDir(path string) ([]os.DirEntry, error)
	Stat(path string) (FileInfo, error)
	Open(path string) (File, error)
}

// WriteFS extends FS atomic content replacement.
type WriteFS interface {
	FS
	Replace(path string, data []byte, perm os.FileMode) error
}

// Filesystem implements WriteFS.
type Filesystem struct{}

// Compile-time validation that Filesystem implements WriteFS.
var _ WriteFS = Filesystem{}

// Open opens path for reading. The returned handle must be closed by the caller.
func (Filesystem) Open(path string) (File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}

	return osFile{file}, nil
}

// Read opens a path and reads at most maxBytes bytes from its contents.
// If more bytes were available, it returns ErrTruncated alongside the bytes that were read.
// Error is returned if the file cannot be read.
func (filesystem Filesystem) Read(path string, maxBytes int64) ([]byte, error) {
	if maxBytes < 0 || maxBytes == int64(^uint64(0)>>1) {
		return nil, fmt.Errorf("read %s: invalid byte limit %d", path, maxBytes)
	}

	file, err := filesystem.Open(path)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()

		return nil, fmt.Errorf("stat %s: %w", path, err)
	}

	if !info.IsRegular {
		_ = file.Close()

		return nil, &os.PathError{Op: "read", Path: path, Err: syscall.EISDIR}
	}

	// Read one extra byte to detect whether the file exceeds maxBytes.
	data, readErr := io.ReadAll(io.LimitReader(file, maxBytes+1))
	closeErr := file.Close()

	if readErr != nil {
		return nil, fmt.Errorf("read %s: %w", path, readErr)
	}

	if closeErr != nil {
		return nil, fmt.Errorf("close %s: %w", path, closeErr)
	}

	truncated := int64(len(data)) > maxBytes
	if truncated {
		data = data[:maxBytes]

		return data, fmt.Errorf("read %s: %w", path, ErrTruncated)
	}

	return data, nil
}

// Replace atomically writes data at a path.
func (Filesystem) Replace(path string, data []byte, perm os.FileMode) error {
	directory := filepath.Dir(path)

	file, err := os.CreateTemp(directory, ".rhc-heartbeat-*")
	if err != nil {
		return &os.PathError{Op: "replace", Path: path, Err: err}
	}

	tempPath := file.Name()
	removeTemp := true

	defer func() {
		_ = file.Close()

		if removeTemp {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := file.Write(data); err != nil {
		return &os.PathError{Op: "replace", Path: path, Err: err}
	}

	if err := file.Chmod(perm.Perm()); err != nil {
		return &os.PathError{Op: "replace", Path: path, Err: err}
	}

	if err := file.Sync(); err != nil {
		return &os.PathError{Op: "replace", Path: path, Err: err}
	}

	if err := file.Close(); err != nil {
		return &os.PathError{Op: "replace", Path: path, Err: err}
	}

	if err := os.Rename(tempPath, path); err != nil {
		return &os.PathError{Op: "replace", Path: path, Err: err}
	}

	removeTemp = false

	return nil
}

type osFile struct{ *os.File }

// Stat returns metadata for the file represented by the opened descriptor.
func (f osFile) Stat() (FileInfo, error) {
	info, err := f.File.Stat()
	if err != nil {
		return FileInfo{}, &os.PathError{Op: "stat", Path: f.Name(), Err: err}
	}

	return metadata(info), nil
}

// metadata converts standard-library file information into the package's
// filesystem metadata representation.
func metadata(info os.FileInfo) FileInfo {
	mode := info.Mode()

	m := FileInfo{
		Mode:      mode,
		UID:       0,
		GID:       0,
		IsDir:     info.IsDir(),
		IsRegular: mode.IsRegular(),
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		m.UID, m.GID = stat.Uid, stat.Gid
	}

	return m
}

// ReadDir returns all entries under a directory.
func (Filesystem) ReadDir(path string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, &os.PathError{Op: "readdir", Path: path, Err: err}
	}

	return entries, nil
}

// Stat returns metadata for the resolved path.
func (Filesystem) Stat(path string) (FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, &os.PathError{Op: "stat", Path: path, Err: err}
	}

	return metadata(info), nil
}
