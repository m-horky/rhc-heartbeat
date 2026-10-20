// Package subman reads subscription-manager data from the local system.
//
// GetFacts executes subscription-manager facts --list through an injected
// command runner and parses its key/value output into a map. Execution and
// parsing failures are returned as errors.
//
// ReadSyspurpose reads the system purpose from subscription-manager's local
// syspurpose file through an injected filesystem. A missing file means that
// no system purpose is set and yields an empty result; unreadable or
// malformed files are returned as errors.
//
//	facts, _ := subman.GetFacts(exec.OSRunner{}, "/usr/libexec/rhc/heartbeat-rhsm")
//	instanceID := facts["aws_instance_id"]
//	_ = instanceID
//
//	syspurpose, _ := subman.ReadSyspurpose(fs.Filesystem{}, "/etc/rhsm/syspurpose/syspurpose.json")
//	role := syspurpose.Role
//	_ = role
package subman
