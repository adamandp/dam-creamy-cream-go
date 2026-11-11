package infrastructure

import "context"

// ImageStore is the storage boundary shared by the users and products modules.
// Returned URLs remain stable across deployments; the MinIO bucket stays private.
type ImageStore interface {
	Upload(ctx context.Context, scope, ownerID string, data []byte) (string, error)
	Delete(ctx context.Context, imageURL string) error
	Get(ctx context.Context, scope, ownerID, filename string) (StoredImage, error)
}

type StoredImage struct {
	Body        []byte
	ContentType string
}
