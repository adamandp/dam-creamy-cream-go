package integration_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/categories"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
)

func TestCategoryRedisCacheAgainstRealServices(t *testing.T) {
	if os.Getenv("REDIS_INTEGRATION") != "1" || os.Getenv("PG_INTEGRATION") != "1" {
		t.Skip("requires disposable Redis and PostgreSQL")
	}
	db := connectPrismaFixture(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://127.0.0.1:6379/0"
	}
	client, err := infrastructure.NewRedis(ctx, redisURL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	key := "categories:v0:1:10"
	if err := client.Del(ctx, key).Err(); err != nil {
		t.Fatal(err)
	}
	defer client.Del(context.Background(), key)
	// No stale shared revision state should leak into the isolated test.
	if err := client.Del(ctx, "categories:revision").Err(); err != nil {
		t.Fatal(err)
	}
	defer client.Del(context.Background(), "categories:revision")
	cat := domain.Category{Name: "Ice Cream", IsActive: true}
	if err := tx.Create(&cat).Error; err != nil {
		t.Fatal(err)
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	categories.NewService(tx, client, sessions.NewService(tx, "test-access", "test-refresh")).Routes(app)
	res, err := app.Test(httptest.NewRequest("GET", "/categories?page=1&limit=10", nil))
	if err != nil {
		t.Fatal(err)
	}
	var first map[string]any
	if err := json.NewDecoder(res.Body).Decode(&first); err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != 200 || first["data"] == nil {
		t.Fatalf("first response invalid: %v", first)
	}
	ttl := client.TTL(ctx, key).Val()
	if ttl <= 0 || ttl > time.Hour {
		t.Fatalf("expected 1h Redis caching, ttl=%v", ttl)
	}
	// If the backend has cached the first result, the next read must serve it.
	if err := tx.Model(&cat).Update("name", "Changed In DB").Error; err != nil {
		t.Fatal(err)
	}
	res, err = app.Test(httptest.NewRequest("GET", "/categories?page=1&limit=10", nil))
	if err != nil {
		t.Fatal(err)
	}
	var cached map[string]any
	if err := json.NewDecoder(res.Body).Decode(&cached); err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	rows, ok := cached["data"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("unexpected Redis-cached shape: %v", cached)
	}
	item, ok := rows[0].(map[string]any)
	if !ok || item["name"] != "Ice Cream" {
		t.Fatalf("cache miss on second request: %v", item)
	}
}
