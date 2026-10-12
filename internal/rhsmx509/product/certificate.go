package product

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/m-horky/rhc-heartbeat/internal/fs"
)

type Product struct {
	ID            int
	Name          string
	Versions      []string
	Architectures []string
	Tags          []string
}

var extPrefix = [9]int{1, 3, 6, 1, 4, 1, 2312, 9, 1} //nolint:gochecknoglobals

const (
	maxCertificateSize = 1024 * 1024 // 1 MiB
	productIndex       = 9
	typeIndex          = 10
	nameSuffix         = 1
	versionSuffix      = 2
	archSuffix         = 3
	tagsSuffix         = 4
)

// Read reads PEM-encoded product certificates and extracts their metadata.
func Read(filesystem fs.FS, directories ...string) ([]Product, error) {
	var products []Product

	for _, dir := range directories {
		files, err := filesystem.ReadDir(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf("read product certificates in %s: %w", dir, err)
		}

		for _, file := range files {
			if file.IsDir() || filepath.Ext(file.Name()) != ".pem" {
				continue
			}

			path := filepath.Join(dir, file.Name())

			raw, err := filesystem.Read(path, maxCertificateSize)
			if err != nil {
				return nil, fmt.Errorf("read product certificate %s: %w", path, err)
			}

			fileProducts, err := readProducts(raw)
			if err != nil {
				return nil, fmt.Errorf("parse product certificate %s: %w", path, err)
			}

			products = append(products, fileProducts...)
		}
	}

	return products, nil
}

// readProducts parses a RHSM Product x509 file and returns
// product included within.
func readProducts(raw []byte) ([]Product, error) {
	block, _ := pem.Decode(raw)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("expected PEM CERTIFICATE block, got %v", block)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("invalid X.509 certificate: %w", err)
	}

	products, err := readProductExtensions(cert.Extensions)
	if err != nil {
		return nil, fmt.Errorf("read product extensions: %w", err)
	}

	return products, nil
}

func readProductExtensions(extensions []pkix.Extension) ([]Product, error) {
	products := make(map[int]Product, 1)

	for _, extension := range extensions {
		id := extension.Id
		if len(id) < len(extPrefix) || !slices.Equal(id[:len(extPrefix)], extPrefix[:]) {
			continue
		}

		if len(id) != typeIndex+1 {
			return nil, fmt.Errorf("invalid product certificate extension ID: %v", id)
		}

		// extract ID and create in-memory representation, if missing
		productID := id[productIndex]

		product, ok := products[productID]
		if !ok {
			product = Product{
				ID:            productID,
				Name:          "",
				Versions:      []string{},
				Architectures: []string{},
				Tags:          []string{},
			}
		}

		// set product information
		switch id[typeIndex] {
		case nameSuffix:
			product.Name = string(extension.Value)
		case versionSuffix:
			product.Versions = strings.Split(string(extension.Value), ",")
		case archSuffix:
			product.Architectures = strings.Split(string(extension.Value), ",")
		case tagsSuffix:
			product.Tags = strings.Split(string(extension.Value), ",")
		}

		products[productID] = product
	}

	var result []Product
	for _, product := range products {
		result = append(result, product)
	}

	slices.SortFunc(result, func(a, b Product) int { return a.ID - b.ID })

	return result, nil
}
