// Package product exposes metadata from the system's Red Hat product certificates.
//
// Get scans /etc/pki/product-default and /etc/pki/product and returns []Product.
// Missing certificate directories are treated as empty; other read or certificate
// parsing errors are returned.
//
//	products, _ := product.Get()
//	for _, p := range products {
//		fmt.Println(p.ID, p.Name, p.Versions, p.Architectures, p.Tags)
//	}
package product
