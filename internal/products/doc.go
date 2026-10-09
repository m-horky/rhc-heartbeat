// Package products lists product IDs from Red Hat product certificates.
//
// ListIDs reads certificate directories through an injected filesystem, making
// certificate discovery independent of direct operating-system calls. Missing
// directories are ignored; other filesystem and certificate errors are returned.
//
//	ids, err := products.ListIDs(
//		fs.Filesystem{},
//		"/etc/pki/product-default",
//		"/etc/pki/product",
//	)
//	if err != nil {
//		return err
//	}
//	_ = ids
package products
