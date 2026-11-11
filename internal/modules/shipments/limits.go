package shipments

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CostRateLimiter protects the upstream provider quota across API replicas.
type CostRateLimiter interface {
	Allow(ctx context.Context, clientIP string) (bool, error)
}

type RedisCostRateLimiter struct {
	client      *redis.Client
	limit       int64
	globalLimit int64
	window      time.Duration
}

// NewRedisCostRateLimiter enforces both per-IP and total provider request caps.
// With no explicit global limit, the global cap matches the per-IP limit.
func NewRedisCostRateLimiter(client *redis.Client, limit int, window time.Duration, globalLimit ...int) *RedisCostRateLimiter {
	global := limit
	if len(globalLimit) > 0 {
		global = globalLimit[0]
	}
	return &RedisCostRateLimiter{client: client, limit: int64(limit), globalLimit: int64(global), window: window}
}

// Both quota checks, increments and initial expirations execute atomically.
// Denied requests consume neither the per-IP budget nor the global budget.
var costLimitScript = redis.NewScript(`
local ipCount = tonumber(redis.call('GET', KEYS[1]) or '0')
local globalCount = tonumber(redis.call('GET', KEYS[2]) or '0')
if ipCount >= tonumber(ARGV[2]) or globalCount >= tonumber(ARGV[3]) then
    return 0
end
local ipNew = redis.call('INCR', KEYS[1])
if ipNew == 1 then
    redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
local globalNew = redis.call('INCR', KEYS[2])
if globalNew == 1 then
    redis.call('PEXPIRE', KEYS[2], ARGV[1])
end
return 1
`)

func (l *RedisCostRateLimiter) Allow(ctx context.Context, clientIP string) (bool, error) {
	if l == nil || l.client == nil || l.limit < 1 || l.globalLimit < 1 || l.window <= 0 {
		return false, fmt.Errorf("shipping quote rate limiter is not configured")
	}
	ipKey := fmt.Sprintf("shipping:quote:rate:v2:ip:%x", sha256.Sum256([]byte(clientIP)))
	result, err := costLimitScript.Run(
		ctx, l.client,
		[]string{ipKey, "shipping:quote:rate:v2:global"},
		l.window.Milliseconds(), l.limit, l.globalLimit,
	).Int64()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}
