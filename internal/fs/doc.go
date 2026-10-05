// Package fs provides filesystem capabilities.
//
// Use FS in consumers and inject an implementation rather than calling
// the operating system directly. Tests can provide a small fake that
// implements the same interface.
//
//
// Read opens and validates the file before returning its contents:
//
//   func loadConfig(filesystem fs.FS, path string) ([]byte, error) {
//        return filesystem.Read(path)
//   }
//
// The File returned by Open is a descriptor-backed read handle and must be
// closed by the caller.
package fs
