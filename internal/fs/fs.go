//go:build linux

package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// FileInfo contains the filesystem information needed by the configuration
// trust policy. Its shape follows the information exposed by os.FileInfo while
// keeping ownership extraction inside this package.
type FileInfo struct {
	Mode      os.FileMode
	UID       uint32
	GID       uint32
	IsDir     bool
	IsRegular bool
	IsSymlink bool
}

// File is the read-only subset of *os.File used by consumers of FS.
type File interface {
	io.Reader
	Stat() (FileInfo, error)
	Close() error
}

// FS is the read-only filesystem contract used by the loader. Read reads a
// validated regular file. ReadDir has the same entry type and ordering
// semantics as os.ReadDir. Stat rejects symlinks. Open rejects symlinks in
// every path component and returns a handle to the opened file.
type FS interface {
	Read(path string) ([]byte, error)
	ReadDir(path string) ([]os.DirEntry, error)
	Stat(path string) (FileInfo, error)
	Open(path string) (File, error)
}

// Filesystem implements FS.
type Filesystem struct{}

// Compile-time validation that Filesystem implements FS.
var _ FS = Filesystem{}

// openHow is the Linux openat2(2) argument. It is kept here instead of
// exposing Linux-specific descriptors to callers of this package.
type openHow struct {
	Flags   uint64
	Mode    uint64
	Resolve uint64
}

const (
	resolveNoSymlinks = 0x04
	resolveBeneath    = 0x08
	atFDCWD           = -100
	sysOpenat2        = 437 // SYS_openat2 on Linux
)

// openat2 invokes Linux's openat2(2) system call and wraps the returned
// descriptor in an *os.File. The path is resolved relative to dirfd and the
// caller supplies the kernel's resolution restrictions.
func openat2(dirfd int, path string, flags int, resolve uint64) (*os.File, error) {
	name, err := syscall.BytePtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("convert path for openat2: %w", err)
	}

	how := openHow{Flags: uint64(flags), Mode: 0, Resolve: resolve}

	fd, _, errno := syscall.Syscall6(
		sysOpenat2,
		uintptr(int64(dirfd)),
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(&how)),
		unsafe.Sizeof(how),
		0,
		0,
	)
	if errno != 0 {
		return nil, errno
	}

	return os.NewFile(fd, path), nil
}

// openReadNoSymlinks opens path for reading without following symlinks in
// either the parent path or the final path component. The final component is
// resolved relative to the opened parent directory descriptor.
func openReadNoSymlinks(path string) (*os.File, error) {
	cleanPath := filepath.Clean(path)
	parent, base := filepath.Dir(cleanPath), filepath.Base(cleanPath)

	// Open the parent without following any symlink, then resolve the final
	// component relative to that directory descriptor. This keeps both the
	// parent traversal and the final open under openat2 policy.
	directory, err := openat2(atFDCWD, parent, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, resolveNoSymlinks)
	if err != nil {
		return nil, err
	}
	defer func() { _ = directory.Close() }()

	return openat2(int(directory.Fd()), base, syscall.O_RDONLY|syscall.O_CLOEXEC, resolveNoSymlinks|resolveBeneath)
}

// Open opens path for reading and rejects symlinks in every path component.
// The returned handle must be closed by the caller.
func (Filesystem) Open(path string) (File, error) {
	file, err := openReadNoSymlinks(path)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}

	return osFile{file}, nil
}

// Read opens path, verifies that it is a regular file, and reads its
// contents.
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

	if !info.IsRegular || info.IsSymlink {
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
// platform-neutral metadata representation.
func metadata(info os.FileInfo) FileInfo {
	mode := info.Mode()

	m := FileInfo{
		Mode:      mode,
		UID:       0,
		GID:       0,
		IsDir:     info.IsDir(),
		IsRegular: mode.IsRegular(),
		IsSymlink: mode&os.ModeSymlink != 0,
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		m.UID, m.GID = stat.Uid, stat.Gid
	}

	return m
}

// ReadDir returns the entries in path in the same sorted order as
// os.ReadDir. It does not follow entries while enumerating them.
func (Filesystem) ReadDir(path string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, &os.PathError{Op: "readdir", Path: path, Err: err}
	}

	return entries, nil
}

// Stat returns metadata for path and rejects symlinks.
func (Filesystem) Stat(path string) (FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return FileInfo{}, &os.PathError{Op: "stat", Path: path, Err: err}
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return FileInfo{}, &os.PathError{Op: "stat", Path: path, Err: syscall.ELOOP}
	}

	return metadata(info), nil
}
