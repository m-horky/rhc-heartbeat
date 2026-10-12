// Package bootid reads the kernel's system boot ID.
package bootid

import (
	"fmt"
	"strings"

	"github.com/m-horky/rhc-heartbeat/internal/fs"
)

const (
	path           = "/proc/sys/kernel/random/boot_id"
	maxBootIDBytes = 64
)

// Read returns kernel boot ID.
func Read(filesystem fs.FS) (string, error) {
	data, err := filesystem.Read(path, maxBootIDBytes)
	if err != nil {
		return "", fmt.Errorf("read kernel boot ID: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}
