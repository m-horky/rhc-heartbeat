// Package product exposes metadata from the system's Red Hat product certificates.
//
// Get scans /etc/pki/product-default and /etc/pki/product and reads product metadata
// from the Red Hat product X.509 extension OID. Missing certificate directories are
// treated as empty; other read or certificate parsing errors are returned.
//
//	products, err := product.Get()
//	if err != nil {
//		return err
//	}
//	for _, p := range products {
//		_ = p.ID
//	}
package product
