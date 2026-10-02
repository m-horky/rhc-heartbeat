// Package fs provides the read-only filesystem capability.
//
// Use FS in consumers and inject an implementation rather than calling
// the operating system directly. Tests can provide a small fake that
// implements the same interface.
//
// Read opens and validates a file before returning its contents:
//
//	func loadConfig(filesystem fs.FS, path string) ([]byte, error) {
//		return filesystem.Read(path)
//	}
//
// Filesystem.Stat rejects symlinks. Filesystem.Open also rejects symlinks in
// every path component. The returned File is a descriptor-backed read handle
// and must be closed by the caller.
//
// FS.ReadDir follows os.ReadDir's behavior: it returns []os.DirEntry in
// lexical order. Directory entries should be inspected before opening a path;
// callers should still treat the filesystem as mutable and handle errors from
// a subsequent Open call.
package fs
