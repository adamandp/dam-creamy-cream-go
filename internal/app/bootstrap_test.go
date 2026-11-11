package app

import (
	"reflect"
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/config"
	"github.com/gofiber/fiber/v3"
)

func TestHTTPTrustedProxyConfiguration(t *testing.T) {
	direct := httpConfig(config.Config{})
	if direct.TrustProxy || direct.ProxyHeader != "" {
		t.Fatalf("unconfigured proxies must not trust forwarded client IPs")
	}

	trusted := []string{"127.0.0.1/32", "192.0.2.0/24"}
	options := httpConfig(config.Config{TrustedProxyCIDRs: trusted})
	if !options.TrustProxy || options.ProxyHeader != fiber.HeaderXForwardedFor {
		t.Fatalf("trusted proxy forwarding is not enabled correctly")
	}
	if !reflect.DeepEqual(options.TrustProxyConfig.Proxies, trusted) {
		t.Fatalf("trusted proxy list mismatch: %v", options.TrustProxyConfig.Proxies)
	}
}
