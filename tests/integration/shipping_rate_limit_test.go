package integration_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/shipments"
)

func TestShippingQuoteRedisRateLimit(t *testing.T) {
	if os.Getenv("REDIS_INTEGRATION") != "1" {
		t.Skip("requires disposable Redis")
	}
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
	if err := client.Del(ctx, "shipping:quote:rate:v2:global").Err(); err != nil {
		t.Fatal(err)
	}
	limiter := shipments.NewRedisCostRateLimiter(client, 3, time.Minute, 100)
	uniqueIP := fmt.Sprintf("198.51.100.77-%d", time.Now().UnixNano())
	for n := 0; n < 4; n++ {
		allowed, err := limiter.Allow(ctx, uniqueIP)
		if err != nil {
			t.Fatal(err)
		}
		if allowed != (n < 3) {
			t.Fatalf("request %d allowed=%t, expected %t", n+1, allowed, n < 3)
		}
	}
	otherIP := uniqueIP + "-other"
	allowed, err := limiter.Allow(ctx, otherIP)
	if err != nil || !allowed {
		t.Fatalf("independent IP must get its own budget: allowed=%t err=%v", allowed, err)
	}

	var successful atomic.Int64
	var wg sync.WaitGroup
	concurrentIP := uniqueIP + "-concurrent"
	for n := 0; n < 20; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, callErr := limiter.Allow(ctx, concurrentIP)
			if callErr != nil {
				t.Errorf("Redis rate limiter: %v", callErr)
				return
			}
			if ok {
				successful.Add(1)
			}
		}()
	}
	wg.Wait()
	if successful.Load() != 3 {
		t.Fatalf("concurrent per-IP quota should allow exactly 3 requests; got %d", successful.Load())
	}

	// The CI Redis instance is disposable. Reset only the dedicated quote
	// global key so this test can exercise two independent quota scenarios.
	const globalKey = "shipping:quote:rate:v2:global"
	if err := client.Del(ctx, globalKey).Err(); err != nil {
		t.Fatal(err)
	}
	defer client.Del(context.Background(), globalKey)

	globalLimiter := shipments.NewRedisCostRateLimiter(client, 3, time.Minute, 2)
	for n, ip := range []string{uniqueIP + "-global-a", uniqueIP + "-global-b", uniqueIP + "-global-c"} {
		allowed, err := globalLimiter.Allow(ctx, ip)
		if err != nil {
			t.Fatal(err)
		}
		if allowed != (n < 2) {
			t.Fatalf("global request %d allowed=%t, expected %t", n+1, allowed, n < 2)
		}
	}
	// A request denied by the global budget must not consume its IP quota.
	raisedLimiter := shipments.NewRedisCostRateLimiter(client, 3, time.Minute, 3)
	allowed, err = raisedLimiter.Allow(ctx, uniqueIP+"-global-c")
	if err != nil || !allowed {
		t.Fatalf("globally rejected requests must not consume per-IP quota: allowed=%t err=%v", allowed, err)
	}

	if err := client.Del(ctx, globalKey).Err(); err != nil {
		t.Fatal(err)
	}
	concurrentGlobal := shipments.NewRedisCostRateLimiter(client, 3, time.Minute, 4)
	successful.Store(0)
	for n := 0; n < 20; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			ip := fmt.Sprintf("%s-global-parallel-%d", uniqueIP, n)
			ok, callErr := concurrentGlobal.Allow(ctx, ip)
			if callErr != nil {
				t.Errorf("Redis global rate limiter: %v", callErr)
				return
			}
			if ok {
				successful.Add(1)
			}
		}(n)
	}
	wg.Wait()
	if successful.Load() != 4 {
		t.Fatalf("concurrent global quota should allow exactly 4 requests; got %d", successful.Load())
	}
}
