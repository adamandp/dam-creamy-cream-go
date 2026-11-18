package health

import (
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Result is the public health status returned by a probe.
type Result struct {
	Status string `json:"status"`
}

// Service provides application health checks independently of HTTP transport.
type Service struct {
	params Params
}

// Params holds dependencies used by the health checks.
type Params struct {
	Database    *gorm.DB
	Redis       *redis.Client
	MinIO       *minio.Client
	MinIOBucket string
}
