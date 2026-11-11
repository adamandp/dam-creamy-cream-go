package categories

import (
	"context"
	"log/slog"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	cache    *redis.Client
	sessions *sessions.Service
}

func NewService(db *gorm.DB, cache *redis.Client, ses *sessions.Service) *Service {
	return &Service{db: db, cache: cache, sessions: ses}
}

const categoryRevisionKey = "categories:revision"

func (s *Service) invalidateCache() {
	// Redis is best-effort: a cache outage must not report a failed category
	// mutation after PostgreSQL has already committed the write.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := s.cache.Incr(ctx, categoryRevisionKey).Err(); err != nil {
		slog.Warn("category cache revision increment failed", "error", err)
	}
}
