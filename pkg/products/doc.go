// Package products exposes product IDs from the system's Red Hat product certificates.
//
// Get scans /etc/pki/product-default and /etc/pki/product. IDs are read from
// the Red Hat product X.509 extension OID, deduplicated, and returned in sorted order.
// Missing certificate directories are treated as empty; other read or certificate
// parsing errors are returned.
//
//	ids, err := products.Get()
//	if err != nil {
//		return err
//	}
//	for _, id := range ids {
//		_ = id
//	}
package products
