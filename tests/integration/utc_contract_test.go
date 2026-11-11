package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
)

// The source Prisma schema uses TIMESTAMP(3) WITHOUT TIME ZONE. Go must
// write UTC clock values, not local wall time, or a WIB host shifts every
// newly created and updated record by seven hours.
func TestProductionGORMUTCClock(t *testing.T) {
	if os.Getenv("PG_INTEGRATION") != "1" {
		t.Skip("requires disposable PostgreSQL")
	}
	db, err := infrastructure.NewPostgres(context.Background(), os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()

	// Must be UTC even when running the backend on a non-UTC machine.
	if got := db.NowFunc(); got.Location() != time.UTC {
		t.Fatalf("GORM clock must use UTC; got %v (%v)", got, got.Location())
	}
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	role := domain.Role{Name: "UTCRegression"}
	if err := tx.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	var stored domain.Role
	if err := tx.First(&stored, "id = ?", role.ID.ID).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, v := range []time.Time{stored.CreatedAt, stored.UpdatedAt} {
		if d := v.Sub(now); d < -10*time.Second || d > 10*time.Second {
			t.Fatalf("stored Prisma timestamp offset from UTC: %v; server: %v", v, now)
		}
	}
}
