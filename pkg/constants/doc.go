// Package constants exposes the system paths and environment variable names
// used by rhc-heartbeat that are not configurable from its configuration file.
//
// Usage of the environment variable overrides is strictly internal and for
// development purposes only.
//
//	config := constants.Path(constants.ConfigPath)
//	lock := constants.Path(constants.LockPath)
package constants
