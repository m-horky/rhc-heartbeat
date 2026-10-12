// Package heartbeat collects host identity and clock readings into heartbeat records.
//
// Get accepts a Kind. System profile data, including marketplace metadata and vCPU
// count, and product certificate IDs are best-effort: read failures are logged and
// the heartbeat is returned without the unavailable fields.
//
//	hb, err := heartbeat.Get(heartbeat.KindPing)
//	if err != nil {
//		return err
//	}
//	_ = hb
package heartbeat
