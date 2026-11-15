package infrastructure

import (
	"context"
	"fmt"
	"time"

	"dam-creamy-cream-go/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

func NewMinIO(
	ctx context.Context,
	cfg config.MinIOConfig,
	log *logrus.Entry,
) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("initialize minio client: %w", err)
	}

	// verify bucket accessibility
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	exists, err := client.BucketExists(checkCtx, cfg.Bucket)
	if err != nil {
		log.WithError(err).Warn("MinIO bucket check failed")
		return client, nil
	}

	// create bucket if it does not exist
	if !exists {
		if err := client.MakeBucket(checkCtx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create minio bucket: %w", err)
		}
	}

	return client, nil
}
