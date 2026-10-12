// Package heartbeat collects boot and clock readings, system profile, product
// information, and consumer identity when available. Missing identity fields do not
// prevent collection, so callers can cache the heartbeat and resolve identity before upload.
//
//	hb, _ := heartbeat.Get(heartbeat.KindPing)
//	if hb.HostID != nil {
//		fmt.Println(*hb.HostID)
//	}
package heartbeat
