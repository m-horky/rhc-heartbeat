// Package bootid reads the kernel's system boot ID.
package bootid

import (
	"fmt"
	"strings"

	"github.com/m-horky/rhc-heartbeat/internal/fs"
)

const path = "/proc/sys/kernel/random/boot_id"

// Read returns the trimmed contents of the kernel boot ID file.
func Read(filesystem fs.FS) (string, error) {
	data, err := filesystem.Read(path)
	if err != nil {
		return "", fmt.Errorf("read kernel boot ID: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}
