//go:build linux

package fs

import (
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

// FS is the filesystem contract used by loaders. Read validates that the
// resolved target is a regular file.
type FS interface {
	Read(path string) ([]byte, error)
	ReadDir(path string) ([]os.DirEntry, error)
	Stat(path string) (FileInfo, error)
	Open(path string) (File, error)
}

// WriteFS extends FS with append and atomic replacement operations.
type WriteFS interface {
	FS
	Append(path string, data []byte, perm os.FileMode) error
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

// Read opens path, verifies that its resolved target is a regular file, and
// reads its contents.
func (filesystem Filesystem) Read(path string) ([]byte, error) {
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

	data, readErr := io.ReadAll(file)
	closeErr := file.Close()

	if readErr != nil {
		return nil, fmt.Errorf("read %s: %w", path, readErr)
	}

	if closeErr != nil {
		return nil, fmt.Errorf("close %s: %w", path, closeErr)
	}

	return data, nil
}

// Append opens path, appends data, and syncs both the file and its containing directory.
func (Filesystem) Append(path string, data []byte, perm os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, perm.Perm())
	if err != nil {
		return &os.PathError{Op: "append", Path: path, Err: err}
	}

	info, err := file.Stat()
	if err == nil && !info.Mode().IsRegular() {
		err = syscall.EINVAL
	}

	if err == nil {
		err = file.Chmod(perm.Perm())
	}

	if err == nil {
		err = writeAll(file, data)
	}

	if err == nil {
		err = file.Sync()
	}

	closeErr := file.Close()

	if err != nil {
		return &os.PathError{Op: "append", Path: path, Err: err}
	}

	if closeErr != nil {
		return &os.PathError{Op: "close", Path: path, Err: closeErr}
	}

	if err := syncDirectory(filepath.Dir(path)); err != nil {
		return &os.PathError{Op: "sync directory", Path: filepath.Dir(path), Err: err}
	}

	return nil
}

// Replace atomically writes data at path.
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

	if err := writeAll(file, data); err != nil {
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

	if err := syncDirectory(directory); err != nil {
		return &os.PathError{Op: "sync directory", Path: directory, Err: err}
	}

	return nil
}

// syncDirectory syncs directory metadata after a filesystem update.
func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open directory: %w", err)
	}

	syncErr := directory.Sync()
	closeErr := directory.Close()

	if syncErr != nil {
		return fmt.Errorf("sync directory: %w", syncErr)
	}

	if closeErr != nil {
		return fmt.Errorf("close directory: %w", closeErr)
	}

	return nil
}

// writeAll writes all data to writer or returns the first write error.
func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := writer.Write(data)
		if err != nil {
			return fmt.Errorf("write data: %w", err)
		}

		if written == 0 {
			return io.ErrShortWrite
		}

		data = data[written:]
	}

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

// ReadDir returns the entries in path in the same sorted order as
// os.ReadDir.
func (Filesystem) ReadDir(path string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, &os.PathError{Op: "readdir", Path: path, Err: err}
	}

	return entries, nil
}

// Stat returns metadata for the resolved target of path.
func (Filesystem) Stat(path string) (FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, &os.PathError{Op: "stat", Path: path, Err: err}
	}

	return metadata(info), nil
}
