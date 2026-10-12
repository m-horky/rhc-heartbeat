// Package heartbeat collects host identity, boot identity, and clock readings into
// heartbeat records.
//
//	hb, err := heartbeat.Get(heartbeat.KindPing)
//	if err != nil {
//		return err
//	}
//	if hb.ProductIDs != nil {
//		fmt.Println(*hb.ProductIDs)
//	}
package heartbeat
