package infrastructure

import (
	"context"
	"fmt"

	"dam-creamy-cream-go/internal/dal/model"

	"gorm.io/gorm"
)

// Migrate uses the same manually maintained model registry as GORM Gen.
// It requires the existing PostgreSQL enum types from the NestJS migrations.
func Migrate(ctx context.Context, db *gorm.DB) error {
	if err := db.WithContext(ctx).AutoMigrate(model.All()...); err != nil {
		return fmt.Errorf("auto-migrate models: %w", err)
	}
	return nil
}
