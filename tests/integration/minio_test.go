package integration_test

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
	"github.com/google/uuid"
)

func TestMinIOUploadReadDelete(t *testing.T) {
	if os.Getenv("MINIO_INTEGRATION") != "1" {
		t.Skip("set MINIO_INTEGRATION=1 and run a disposable MinIO server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()
	endpoint := os.Getenv("MINIO_TEST_ENDPOINT")
	if endpoint == "" {
		endpoint = "127.0.0.1:9000"
	}
	store, err := infrastructure.NewMinIO(ctx, endpoint, "minioadmin", "minioadmin", "creamy-cream-test", "http://localhost:3000", false)
	if err != nil {
		t.Fatal(err)
	}
	ownerID := uuid.NewString()
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0, 0, 0, 0}
	imageURL, err := store.Upload(ctx, "products", ownerID, jpeg)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(imageURL)
	if err != nil || parsed.Host != "localhost:3000" || !strings.HasPrefix(parsed.Path, "/media/products/"+ownerID+"/") {
		t.Fatalf("unexpected media URL: %s (%v)", imageURL, err)
	}
	filename := strings.TrimPrefix(parsed.Path, "/media/products/"+ownerID+"/")
	actual, err := store.Get(ctx, "products", ownerID, filename)
	if err != nil {
		t.Fatal(err)
	}
	if actual.ContentType != "image/jpeg" || !bytes.Equal(actual.Body, jpeg) {
		t.Fatalf("minio roundtrip mismatch: %+v", actual)
	}
	if err := store.Delete(ctx, "https://res.cloudinary.com/old/image.jpg"); err != nil {
		t.Fatalf("legacy image URL must not be deleted: %v", err)
	}
	if err := store.Delete(ctx, imageURL); err != nil {
		t.Fatal(err)
	}
	_, err = store.Get(ctx, "products", ownerID, filename)
	if !errors.Is(err, infrastructure.ErrImageNotFound) {
		t.Fatalf("expected object gone, got %v", err)
	}
}
