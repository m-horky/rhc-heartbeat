// Package rhsm loads and resolves subscription-manager configuration.
// Load merges the embedded RHSM defaults with values from the supplied INI
// file, then returns server, proxy, and CA settings in Config. An absent file
// leaves the defaults in effect; malformed or invalid settings return an error.
//
// For example, load the system RHSM configuration and use the resolved server
// and CA settings:
//
//	filesystem := fs.Filesystem{}
//	cfg, err := rhsm.Load(filesystem, "/etc/rhsm/rhsm.conf")
//	if err != nil {
//		return err
//	}
//	_ = cfg.Server.Hostname
//	_ = cfg.RHSM.RepoCACert
package rhsm
