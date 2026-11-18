package products

import "time"

const (
	resourceName = "Product"

	imagePrefix       = "products"
	uploadPrefix      = "pending/products"
	presignExpiration = 10 * time.Minute
	cleanupTimeout    = 10 * time.Second
	maxImageSize      = 2 * 1024 * 1024

	sortAscending  = "asc"
	sortDescending = "desc"
)
