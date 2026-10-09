package rhsm

import _ "embed"

// defaultConfig contains the package's embedded configuration defaults.
//
//go:embed default.conf
var defaultConfig []byte

// defaults decodes the embedded RHSM defaults into a partial configuration.
func defaults() (dtoConfig, error) {
	return parse(defaultConfig)
}
