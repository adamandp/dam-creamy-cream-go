package integration_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/categories"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
)

func TestCategoryCacheRevisionAfterAllMutations(t *testing.T) {
	if os.Getenv("REDIS_INTEGRATION") != "1" || os.Getenv("PG_INTEGRATION") != "1" {
		t.Skip("requires disposable Redis and PostgreSQL")
	}
	db := connectPrismaFixture(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	ctx := context.Background()
	cache, err := infrastructure.NewRedis(ctx, os.Getenv("TEST_REDIS_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer cache.Close()
	// New isolated test instance must start without a cached revision.
	keys, err := cache.Keys(ctx, "categories:*").Result()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) > 0 {
		if err := cache.Del(ctx, keys...).Err(); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		keys, _ := cache.Keys(ctx, "categories:*").Result()
		if len(keys) > 0 {
			_ = cache.Del(ctx, keys...).Err()
		}
	}()
	role := domain.Role{Name: "Admin"}
	if err := tx.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := domain.User{RoleID: role.ID.ID, Username: "CacheRevisionAdmin", FullName: "Cache Revision Admin", Password: "fixture", IsActive: true}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	first := domain.Category{Name: "Vanilla Cream", IsActive: true}
	second := domain.Category{Name: "Strawberry Cream", IsActive: true}
	if err := tx.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	ses := sessions.NewService(tx, "cache-rev-access", "cache-rev-refresh")
	tokens, err := ses.NewTokens(ctx, user.ID.ID, role.ID.ID, "", "Test", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	// Two instances sharing the same Redis revision must see each other's writes.
	inst2 := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	categories.NewService(tx, cache, ses).Routes(app)
	categories.NewService(tx, cache, ses).Routes(inst2)
	revision := func(want int64) {
		t.Helper()
		raw, err := cache.Get(ctx, "categories:revision").Result()
		if want == 0 {
			if err == nil && raw == "0" {
				return
			}
			if err != nil {
				return
			} // no revision yet
		}
		if err != nil {
			t.Fatalf("revision read failed: %v", err)
		}
		got, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || got != want {
			t.Fatalf("revision=%v expected %d", raw, want)
		}
	}
	readPages := func(router *fiber.App, size int) map[string]bool {
		t.Helper()
		names := map[string]bool{}
		for page := 1; page <= size; page++ {
			path := fmt.Sprintf("/categories?page=%d&limit=1", page)
			payload := testBody(t, testCall(t, router, "GET", path, "", "", 200))
			rows := payload["data"].([]any)
			if len(rows) != 1 {
				t.Fatalf("page %d missing category: %+v", page, payload)
			}
			names[rows[0].(map[string]any)["name"].(string)] = true
		}
		return names
	}
	before := readPages(app, 2)
	if !before["Vanilla Cream"] || !before["Strawberry Cream"] {
		t.Fatalf("unexpected initial cache %v", before)
	}
	revision(0)
	testBody(t, testCallAs(t, app, tokens.AccessToken, "PATCH", "/categories/"+first.ID.ID, `{"name":"Vanilla Classic"}`, "", 200))
	revision(1)
	after := readPages(inst2, 2)
	if after["Vanilla Cream"] || !after["Vanilla Classic"] {
		t.Fatalf("stale renamed category cache %v", after)
	}
	testBody(t, testCallAs(t, app, tokens.AccessToken, "POST", "/categories", `{"name":"Chocolate Cream","description":null,"isActive":true}`, "", 201))
	revision(2)
	names := readPages(inst2, 3)
	if !names["Chocolate Cream"] || len(names) != 3 {
		t.Fatalf("stale category creation %v", names)
	}
	var third domain.Category
	if err := tx.Where("name = ?", "Chocolate Cream").First(&third).Error; err != nil {
		t.Fatal(err)
	}
	testBody(t, testCallAs(t, app, tokens.AccessToken, "DELETE", "/categories/"+third.ID.ID, "", "", 200))
	revision(3)
	last := readPages(inst2, 2)
	if last["Chocolate Cream"] || len(last) != 2 {
		t.Fatalf("stale category deletion %v", last)
	}
}
