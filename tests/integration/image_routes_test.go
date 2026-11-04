package integration_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/media"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/products"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/users"
	"github.com/gofiber/fiber/v3"
)

func multipartImage(t *testing.T, app *fiber.App, method, path string, payload []byte) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := textproto.MIMEHeader{
		"Content-Disposition": {`form-data; name="file"; filename="photo.jpg"`},
		"Content-Type":        {"image/jpeg"},
	}
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(method, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		result, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s %s returned %d: %s", method, path, resp.StatusCode, string(result))
	}
}

func TestImageRoutesWithRealMinIOAndPostgres(t *testing.T) {
	if os.Getenv("MINIO_INTEGRATION") != "1" || os.Getenv("PG_INTEGRATION") != "1" {
		t.Skip("requires disposable PostgreSQL and MinIO")
	}
	db := connectPrismaFixture(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	endpoint := os.Getenv("MINIO_TEST_ENDPOINT")
	if endpoint == "" {
		endpoint = "127.0.0.1:9000"
	}
	store, err := infrastructure.NewMinIO(ctx, endpoint, "minioadmin", "minioadmin", "creamy-cream-api-test", "http://localhost:3000", false)
	if err != nil {
		t.Fatal(err)
	}
	role := domain.Role{Name: "Customer"}
	if err := tx.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := domain.User{RoleID: role.ID.ID, Username: "Fixture", FullName: "Fixture Name", Password: "test-only", IsActive: true}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	category := domain.Category{Name: "Ice Cream", IsActive: true}
	if err := tx.Create(&category).Error; err != nil {
		t.Fatal(err)
	}
	product := domain.Product{CategoryID: category.ID.ID, Name: "Vanilla", Price: 10000, IsActive: true}
	if err := tx.Create(&product).Error; err != nil {
		t.Fatal(err)
	}

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	users.NewService(tx, sessions.NewService(tx, "access-fixture", "refresh-fixture"), store).Routes(app)
	products.NewService(tx, store).Routes(app)
	media.NewService(store).Routes(app)

	jpg := []byte{0xff, 0xd8, 0xff, 0xe0, 0, 0, 0, 0}
	multipartImage(t, app, "PUT", "/users/"+user.ID.ID+"/image", jpg)
	multipartImage(t, app, "PATCH", "/products/"+product.ID.ID+"/image", jpg)

	var newUser domain.User
	if err := tx.First(&newUser, "id = ?", user.ID.ID).Error; err != nil {
		t.Fatal(err)
	}
	var newProduct domain.Product
	if err := tx.First(&newProduct, "id = ?", product.ID.ID).Error; err != nil {
		t.Fatal(err)
	}
	if newUser.ImageURL == nil || newProduct.ImageURL == nil {
		t.Fatal("image URL not persisted to PostgreSQL")
	}
	for _, imageURL := range []string{*newUser.ImageURL, *newProduct.ImageURL} {
		location, err := url.Parse(imageURL)
		if err != nil {
			t.Fatal(err)
		}
		if location.Host != "localhost:3000" || !strings.HasPrefix(location.Path, "/media/") {
			t.Fatalf("unexpected MinIO media URL %s", imageURL)
		}
		req := httptest.NewRequest("GET", location.Path, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil || resp.StatusCode != 200 || !bytes.Equal(raw, jpg) {
			t.Fatalf("media response status=%d err=%v bytes=%x", resp.StatusCode, err, raw)
		}
		defer store.Delete(context.Background(), imageURL)
	}

	old := *newProduct.ImageURL
	multipartImage(t, app, "PATCH", "/products/"+product.ID.ID+"/image", jpg)
	var latest domain.Product
	if err := tx.First(&latest, "id = ?", product.ID.ID).Error; err != nil {
		t.Fatal(err)
	}
	if latest.ImageURL == nil || *latest.ImageURL == old {
		t.Fatal("replacement must produce a new immutable image URL")
	}
	defer store.Delete(context.Background(), *latest.ImageURL)
	oldPath, _ := url.Parse(old)
	resp, err := app.Test(httptest.NewRequest("GET", oldPath.Path, nil))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != 404 {
		t.Fatalf("old image must be deleted after successful DB update, got %d", resp.StatusCode)
	}
}
