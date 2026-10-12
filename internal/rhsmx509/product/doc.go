// Package product reads product metadata from Red Hat product certificates.
//
// Read accepts an injected filesystem and certificate directories, and returns
// Product values. Missing directories are ignored; other filesystem and
// certificate errors are returned.
//
//	products, err := product.Read(
//		fs.Filesystem{},
//		"/etc/pki/product-default",
//		"/etc/pki/product",
//	)
//	if err != nil {
//		return err
//	}
//	for _, p := range products {
//		fmt.Println(p.ID, p.Name)
//	}
package product
