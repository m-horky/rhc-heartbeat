// Package heartbeat collects host identity and clock readings into heartbeat records.
//
// Get accepts a context and one of KindOn, KindOff, or KindPing. It returns an error for an invalid
// kind, a canceled context, or a failure reading the consumer identity, boot ID, or clocks. System
// profile data, including marketplace metadata and vCPU count, and product certificate IDs are
// best-effort: read failures are logged and the heartbeat is returned without the unavailable fields.
// Heartbeat timestamps include monotonic and boottime durations since boot, plus a UTC wall-clock timestamp.
//
//	hb, err := heartbeat.Get(ctx, heartbeat.KindPing)
//	if err != nil {
//		return err
//	}
//	_ = hb
package heartbeat
