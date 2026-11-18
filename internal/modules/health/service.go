package health

import (
	"context"
	"time"
)

func NewService(params Params) *Service {
	return &Service{params: params}
}

// CheckGo reports liveness without querying external dependencies. Dependency
// outages should not cause restarts of an otherwise healthy application.
func (s *Service) CheckGo() Result {
	return Result{Status: "ok"}
}

// CheckDB reports readiness with a bounded database connectivity check.
func (s *Service) CheckDB(ctx context.Context) Result {
	// limit the database probe duration
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if s.params.Database == nil {
		return Result{Status: "unavailable"}
	}

	// check connectivity directly through the existing SQL pool
	pool, err := s.params.Database.DB()

	if err != nil || pool.PingContext(ctx) != nil {
		return Result{Status: "unavailable"}
	}

	return Result{Status: "ok"}
}

// CheckRedis checks connectivity using the existing Redis client.
func (s *Service) CheckRedis(ctx context.Context) Result {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if s.params.Redis == nil || s.params.Redis.Ping(ctx).Err() != nil {
		return Result{Status: "unavailable"}
	}

	return Result{Status: "ok"}
}

// CheckMinIO checks access to the configured bucket without modifying it.
func (s *Service) CheckMinIO(ctx context.Context) Result {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if s.params.MinIO == nil || s.params.MinIOBucket == "" {
		return Result{Status: "unavailable"}
	}

	exists, err := s.params.MinIO.BucketExists(ctx, s.params.MinIOBucket)

	if err != nil || !exists {
		return Result{Status: "unavailable"}
	}

	return Result{Status: "ok"}
}
