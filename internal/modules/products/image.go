package products

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"dam-creamy-cream-go/internal/exceptions"
	"dam-creamy-cream-go/internal/utils"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

func (s *Service) PresignImage(ctx context.Context, id string, req PresignImageRequest) (*PresignImageResponse, error) {
	product, err := s.ValidateProduct(ctx, id)
	if err != nil {
		return nil, err
	}

	extension := imageExtension(req.ContentType)
	if extension == "" {
		return nil, exceptions.BadRequest("Only image files are allowed!")
	}

	imageKey := path.Join(
		uploadPrefix,
		product.ID,
		uuid.NewString()+extension,
	)

	uploadURL, err := s.minio.PresignedPutObject(
		ctx,
		s.bucket,
		imageKey,
		presignExpiration,
	)

	if err != nil {
		s.log.WithError(err).Error("Failed to generate presigned image URL")
		return nil, exceptions.InternalServerError()
	}

	return &PresignImageResponse{
		ProductID: product.ID,
		UploadURL: uploadURL.String(),
		ImageKey:  imageKey,
		ExpiresIn: int(presignExpiration.Seconds()),
	}, nil
}

func (s *Service) publishImage(ctx context.Context, id string, imageKey string) (string, error) {
	if !validImageKey(imageKey, uploadPrefix, id) {
		return "", exceptions.BadRequest("Invalid product image key")
	}

	info, err := s.minio.StatObject(
		ctx,
		s.bucket,
		imageKey,
		minio.StatObjectOptions{},
	)

	if err != nil {
		switch minio.ToErrorResponse(err).Code {
		case "NoSuchKey", "NoSuchObject", "NotFound":
			return "", exceptions.BadRequest("Product image has not been uploaded")
		}

		s.log.WithError(err).Error("Failed to verify product image")

		return "", exceptions.InternalServerError()
	}

	if info.Size <= 0 || info.Size > maxImageSize {
		return "", exceptions.BadRequest("Image size must not exceed 2 MB")
	}

	object, err := s.minio.GetObject(
		ctx,
		s.bucket,
		imageKey,
		minio.GetObjectOptions{},
	)

	if err != nil {
		s.log.WithError(err).Error("Failed to read uploaded image")
		return "", exceptions.InternalServerError()
	}

	defer object.Close()

	data, err := io.ReadAll(
		io.LimitReader(object, maxImageSize+1),
	)

	if err != nil {
		s.log.WithError(err).Error("Failed to read image content")
		return "", exceptions.InternalServerError()
	}

	if len(data) == 0 || len(data) > maxImageSize {
		return "", exceptions.BadRequest("Image size must not exceed 2 MB")
	}

	contentType := http.DetectContentType(data)
	extension := imageExtension(contentType)

	if extension == "" || path.Ext(imageKey) != extension {
		return "", exceptions.BadRequest("Only valid image files are allowed!")
	}

	objectKey := path.Join(
		imagePrefix,
		id,
		uuid.NewString()+extension,
	)

	imageURL := s.minio.EndpointURL().
		JoinPath(s.bucket, objectKey).
		String()

	if len(imageURL) > 255 {
		s.log.Error("Product image URL exceeds database column length")
		return "", exceptions.InternalServerError()
	}

	_, err = s.minio.PutObject(
		ctx,
		s.bucket,
		objectKey,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)

	if err != nil {
		s.log.WithError(err).Error("Failed to publish product image")
		return "", exceptions.BadRequest(
			utils.UploadErrorMessage(resourceName),
		)
	}

	return imageURL, nil
}

func (s *Service) removeImage(ctx context.Context, id string, imageURL string) {
	baseURL := s.minio.EndpointURL().
		JoinPath(s.bucket).
		String()

	objectKey, ok := strings.CutPrefix(
		imageURL,
		strings.TrimRight(baseURL, "/")+"/",
	)

	if !ok {
		s.log.WithField("imageUrl", imageURL).
			Warn("Skipped image outside managed MinIO storage")

		return
	}

	if !validImageKey(objectKey, imagePrefix, id) {
		s.log.WithField("objectKey", objectKey).
			Warn("Skipped invalid product image key")

		return
	}

	s.removeObject(ctx, objectKey)
}

func (s *Service) removeObject(ctx context.Context, objectKey string) {
	cleanupCtx, cancel := context.WithTimeout(
		context.WithoutCancel(ctx),
		cleanupTimeout,
	)

	defer cancel()

	if err := s.minio.RemoveObject(
		cleanupCtx,
		s.bucket,
		objectKey,
		minio.RemoveObjectOptions{},
	); err != nil {
		s.log.WithError(err).
			WithField("objectKey", objectKey).
			Error("Failed to remove product image")
	}
}

func imageExtension(contentType string) string {
	switch contentType {
	case "image/jpeg", "image/jpg":
		return ".jpg"

	case "image/png":
		return ".png"

	case "image/webp":
		return ".webp"

	default:
		return ""
	}
}

func validImageKey(objectKey, prefix, id string) bool {
	expectedPrefix := fmt.Sprintf("%s/%s/", prefix, id)

	filename, ok := strings.CutPrefix(
		objectKey,
		expectedPrefix,
	)

	if !ok {
		return false
	}

	extension := path.Ext(filename)

	switch extension {
	case ".jpg", ".png", ".webp":
	default:
		return false
	}

	name := strings.TrimSuffix(filename, extension)

	parsed, err := uuid.Parse(name)
	if err != nil {
		return false
	}

	return parsed.String() == name
}
