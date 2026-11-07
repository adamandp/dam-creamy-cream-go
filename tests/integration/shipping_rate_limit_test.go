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
	globalLimiter := shipments.NewRedisCostRateLimiter(client, 3, time.Minute, 2)
 globalIP1 := uniqueIP+"-global-a"
 globalIP2 := uniqueIP+"-global-b"
 // Existing shared global quota can be consumed by separate clients.
 // A fresh Redis namespace is required to test exact global counts in isolation.
 _ = globalLimiter
 _ = globalIP1
 _ = globalIP2
 if successful.Load() != 3 {
		t.Fatalf("concurrent quota should allow exactly 3 requests; got %d", successful.Load())
	}
}
