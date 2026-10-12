package rhsm

import _ "embed"

// defaultConfig contains the package's embedded configuration newPartialFromDefaults.
//
//go:embed default.conf
var defaultConfig []byte

// newPartialFromDefaults decodes the embedded newPartialFromDefaults into a partial configuration.
func newPartialFromDefaults() (*dtoConfig, error) {
	return newPartialFromINI(defaultConfig)
}
