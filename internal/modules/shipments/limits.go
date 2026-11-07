package shipments

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CostRateLimiter protects the upstream provider's quota across API replicas.
type CostRateLimiter interface {
	Allow(ctx context.Context, clientIP string) (bool, error)
}

type RedisCostRateLimiter struct {
	client *redis.Client
	limit  int64
	window time.Duration
}

func NewRedisCostRateLimiter(client *redis.Client, limit int, window time.Duration) *RedisCostRateLimiter {
	return &RedisCostRateLimiter{client: client, limit: int64(limit), window: window}
}

// INCR and first-use expiration must be atomic. Otherwise concurrent calls
// could create a permanently unexpired limit key or evade the first window.
var costLimitScript = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
    redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
return count
`)

func (l *RedisCostRateLimiter) Allow(ctx context.Context, clientIP string) (bool, error) {
	if l == nil || l.client == nil || l.limit < 1 || l.window <= 0 {
		return false, fmt.Errorf("shipping quote rate limiter is not configured")
	}
	// Hash the address so IPs are neither stored directly in Redis keys nor
	// vulnerable to a caller injecting Redis key delimiters.
	key := fmt.Sprintf("shipping:quote:rate:v1:%x", sha256.Sum256([]byte(clientIP)))
	count, err := costLimitScript.Run(ctx, l.client, []string{key}, l.window.Milliseconds()).Int64()
	if err != nil {
		return false, err
	}
	return count <= l.limit, nil
}
