package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// These tests run only on an isolated PostgreSQL with the application schema.
// Never point TEST_DATABASE_URL at production.
func connectPrismaFixture(t *testing.T) *gorm.DB {
	t.Helper()
	if os.Getenv("PG_INTEGRATION") != "1" {
		t.Skip("requires PG_INTEGRATION=1 with disposable Prisma database")
	}
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL must point at the disposable schema")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := infrastructure.NewPostgres(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}
func TestPostgresGORMColumnMapping(t *testing.T) {
	db := connectPrismaFixture(t)
	models := []any{
		&domain.Role{}, &domain.User{}, &domain.UserToken{}, &domain.Address{}, &domain.Category{},
		&domain.Product{}, &domain.Inventory{}, &domain.Cart{}, &domain.CartItem{}, &domain.Order{},
		&domain.OrderVoucher{}, &domain.OrderItem{}, &domain.Payment{}, &domain.Shipment{},
		&domain.Review{}, &domain.Discount{}, &domain.ProductDiscount{}, &domain.ProductInformation{},
		&domain.Voucher{}, &domain.UserVoucher{}, &domain.ProductAnalytics{},
		&domain.Notification{}, &domain.UserNotification{},
	}
	for _, model := range models {
		parsed, err := schema.Parse(model, &sync.Map{}, schema.NamingStrategy{})
		if err != nil {
			t.Fatal(err)
		}
		t.Run(parsed.Table, func(t *testing.T) {
			var columns []string
			if err := db.Raw("SELECT column_name FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = ?", parsed.Table).Scan(&columns).Error; err != nil {
				t.Fatal(err)
			}
			if len(columns) == 0 {
				t.Fatalf("Prisma table %s not found", parsed.Table)
			}
			existing := make(map[string]bool, len(columns))
			for _, column := range columns {
				existing[column] = true
			}
			mapped := map[string]bool{}
			for _, field := range parsed.Fields {
				if field.DBName == "" {
					continue
				}
				mapped[field.DBName] = true
				if !existing[field.DBName] {
					t.Errorf("GORM field %s incorrectly maps to missing DB column %q", field.Name, field.DBName)
				}
			}
			for _, column := range columns {
				if !mapped[column] {
					t.Errorf("Prisma DB column %q not modeled in GORM", column)
				}
			}
		})
	}
}

func TestRealPostgresSessionsRoundTrip(t *testing.T) {
	db := connectPrismaFixture(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	role := domain.Role{Name: "Admin"}
	if err := tx.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	name := "prismatest" + uuid.NewString()[:8]
	user := domain.User{RoleID: role.ID.ID, Username: name, FullName: "Fixture User", Password: "test-only-placeholder", IsActive: true}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	session := sessions.NewService(tx, "access-testing-key", "refresh-testing-key")
	tokens, err := session.NewTokens(context.Background(), user.ID.ID, role.ID.ID, "", "Go integration test", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		t.Fatal("empty JWT")
	}
	var persisted domain.UserToken
	if err := tx.Where("token = ?", tokens.RefreshToken).First(&persisted).Error; err != nil {
		t.Fatal(err)
	}
	var info map[string]string
	if err := json.Unmarshal([]byte(persisted.BrowserInfo), &info); err != nil {
		t.Fatal(err)
	}
	if info["browser"] == "" || info["os"] == "" || info["device"] == "" || info["userAgent"] != "" {
		t.Fatalf("browser information contract: %v", info)
	}
	if persisted.BrowserInfo == "" || persisted.ExpiredAt.Before(time.Now()) {
		t.Errorf("session fields failed to roundtrip: %+v", persisted)
	}
	issued, err := session.Refresh(context.Background(), tokens.RefreshToken)
	if err != nil || issued == "" {
		t.Fatalf("refresh failed: %v", err)
	}
	if _, err := session.VerifyAccess(issued); err != nil {
		t.Errorf("refreshed access token invalid: %v", err)
	}
	if err := session.Logout(context.Background(), tokens.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := session.Refresh(context.Background(), tokens.RefreshToken); err == nil {
		t.Fatal("revoked refresh token is still accepted")
	}
	if err := tx.Where("id = ?", persisted.ID.ID).First(&persisted).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("session should have been deleted, got %v", err)
	}
}
