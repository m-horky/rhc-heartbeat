// Package subman reads subscription-manager facts from the local system.
//
// Read executes subscription-manager facts --list through an injected command
// runner and parses its key/value output into a map. Execution and parsing
// failures are returned as errors.
//
//	facts, err := subman.Read(exec.OSRunner{})
//	if err != nil {
//		return err
//	}
//	instanceID := facts["aws_instance_id"]
//	_ = instanceID
package subman
