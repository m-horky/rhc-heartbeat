package subman

import (
	"encoding/json"
	"errors"
	"fmt"
	iofs "io/fs"
	"strings"

	"github.com/m-horky/rhc-heartbeat/internal/fs"
)

const maxSyspurposeSize = 1024 // 1 KiB

// Syspurpose contains the system purpose attributes managed by
// subscription-manager. Attributes that are not set are nil.
type Syspurpose struct {
	Role         *string
	ServiceLevel *string
	Usage        *string
}

// syspurposeFile mirrors the JSON structure of syspurpose.json. Fields that
// are not collected, such as the deprecated addons, are ignored on decode.
type syspurposeFile struct {
	Role         string `json:"role"`
	ServiceLevel string `json:"service_level_agreement"`
	Usage        string `json:"usage"`
}

// ReadSyspurpose reads the system purpose from subscription-manager's local
// syspurpose file. A missing file means that no system purpose is set and
// yields an empty Syspurpose; unreadable or malformed files are errors.
func ReadSyspurpose(filesystem fs.FS, path string) (Syspurpose, error) {
	data, err := filesystem.Read(path, maxSyspurposeSize)
	if err != nil {
		if errors.Is(err, iofs.ErrNotExist) {
			return Syspurpose{}, nil
		}

		return Syspurpose{}, fmt.Errorf("read system purpose %s: %w", path, err)
	}

	var file syspurposeFile
	if err := json.Unmarshal(data, &file); err != nil {
		return Syspurpose{}, fmt.Errorf("parse system purpose %s: %w", path, err)
	}

	return Syspurpose{
		Role:         newNonEmpty(file.Role),
		ServiceLevel: newNonEmpty(file.ServiceLevel),
		Usage:        newNonEmpty(file.Usage),
	}, nil
}

// newNonEmpty returns a pointer to the trimmed value, or nil when it is blank.
func newNonEmpty(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return new(trimmed)
}
