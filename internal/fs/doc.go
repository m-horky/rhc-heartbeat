// Package fs provides filesystem capabilities.
//
// Use FS in consumers and inject an implementation rather than calling
// the operating system directly. Tests can provide a small fake that
// implements the same interface.
//
// Read opens the file and returns at most the requested number of bytes.
// Callers can use errors.Is to detect truncation:
//
//	data, err := filesystem.Read(path, maxFileSize)
//	if errors.Is(err, fs.ErrTruncated) {
//		slog.Debug("file truncated", "path", path)
//	} else if err != nil {
//		return nil, err
//	}
//	return data, nil
//
// The File returned by Open is a descriptor-backed read handle and must be
// closed by the caller.
package fs
