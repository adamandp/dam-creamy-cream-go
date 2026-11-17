package infrastructure_test

import (
	"context"
	"os"
	"testing"

	"dam-creamy-cream-go/internal/dal/model"
	"dam-creamy-cream-go/internal/dal/query"
	"dam-creamy-cream-go/internal/infrastructure"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestAutoMigrateAgainstPrismaFixture(t *testing.T) {
	dsn := os.Getenv("GORM_MIGRATION_TEST_DSN")
	if dsn == "" {
		t.Skip("requires an isolated PostgreSQL database and GORM_MIGRATION_TEST_DSN")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		pool, err := db.DB()
		if err == nil {
			_ = pool.Close()
		}
	}()
	ctx := context.Background()
	if err := infrastructure.Migrate(ctx, db); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	c := query.Use(db).Category
	active := false
	category := &model.Category{
		Name: "GORM migration test", IsActive: &active,
	}
	if err := c.WithContext(ctx).Create(category); err != nil {
		t.Fatalf("create category: %v", err)
	}
	id := category.ID
	if _, err := uuid.Parse(id); err != nil {
		t.Fatalf("expected BeforeCreate-generated UUID, got %q: %v", id, err)
	}
	t.Cleanup(func() {
		_, _ = c.WithContext(ctx).Where(c.ID.Eq(id)).Delete()
	})
	got, err := c.WithContext(ctx).Where(c.ID.Eq(id)).First()
	if err != nil {
		t.Fatalf("typed query: %v", err)
	}
	if got.IsActive == nil || *got.IsActive {
		t.Fatalf("expected explicit false, got %+v", got.IsActive)
	}
	// Verify the Prisma-style typed select/filter API requested by the caller.
	selected, err := c.WithContext(ctx).
		Select(c.ID, c.Name, c.IsActive).
		Where(c.IsActive.Is(false)).
		Find()
	if err != nil {
		t.Fatalf("typed select/filter: %v", err)
	}
	found := false
	for _, row := range selected {
		if row.ID == id && row.Name == category.Name {
			found = true
		}
	}
	if !found {
		t.Fatal("typed select/filter did not return the inserted category")
	}
}
