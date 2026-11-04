package unit_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/media"
	"github.com/gofiber/fiber/v3"
)

func TestImageFormat(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		mime string
		ext  string
		err  error
	}{
		{"jpeg", []byte{0xff, 0xd8, 0xff, 0xe0}, "image/jpeg", "jpg", nil},
		{"png", []byte{137, 80, 78, 71, 13, 10, 26, 10, 0}, "image/png", "png", nil},
		{"webp", []byte{'R', 'I', 'F', 'F', 12, 0, 0, 0, 'W', 'E', 'B', 'P', 'V', 'P', '8', ' '}, "image/webp", "webp", nil},
		{"text", []byte("<script>alert(1)</script>"), "", "", infrastructure.ErrInvalidImage},
		{"empty", nil, "", "", infrastructure.ErrInvalidImage},
		{"oversized", make([]byte, infrastructure.MaxImageBytes+1), "", "", infrastructure.ErrImageTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mime, ext, err := infrastructure.ImageFormat(tc.data)
			if mime != tc.mime || ext != tc.ext || !errors.Is(err, tc.err) {
				t.Fatalf("ImageFormat = (%q,%q,%v), want (%q,%q,%v)", mime, ext, err, tc.mime, tc.ext, tc.err)
			}
		})
	}
}

type fakeStore struct {
	img infrastructure.StoredImage
	err error
}

func (f fakeStore) Upload(context.Context, string, string, []byte) (string, error) {
	return "", nil
}
func (f fakeStore) Delete(context.Context, string) error { return nil }
func (f fakeStore) Get(context.Context, string, string, string) (infrastructure.StoredImage, error) {
	return f.img, f.err
}

func TestMediaReadUsesPrivateStore(t *testing.T) {
	body := []byte{0xff, 0xd8, 0xff, 0xe0}
	app := fiber.New()
	media.NewService(fakeStore{img: infrastructure.StoredImage{Body: body, ContentType: "image/jpeg"}}).Routes(app)
	req := httptest.NewRequest("GET", "/media/products/5918248d-8523-4351-a9ce-c3e56c6c931c/6c9f7291-04d5-44b1-931a-7634a33c2b21.jpg", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	actual, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || !bytes.Equal(actual, body) {
		t.Fatalf("unexpected status/body: %d %x", resp.StatusCode, actual)
	}
	if resp.Header.Get("Content-Type") != "image/jpeg" ||
		resp.Header.Get("X-Content-Type-Options") != "nosniff" ||
		!strings.Contains(resp.Header.Get("Cache-Control"), "immutable") {
		t.Fatalf("missing media security/cache headers: %v", resp.Header)
	}
}

func TestMediaMissingObject(t *testing.T) {
	app := fiber.New()
	media.NewService(fakeStore{err: infrastructure.ErrImageNotFound}).Routes(app)
	resp, err := app.Test(httptest.NewRequest("GET", "/media/users/5918248d-8523-4351-a9ce-c3e56c6c931c/6c9f7291-04d5-44b1-931a-7634a33c2b21.jpg", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}
