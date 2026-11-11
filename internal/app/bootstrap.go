package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/config"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/addresses"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/auth"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/carts"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/categories"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/discounts"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/health"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/inventory"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/media"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/notifications"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/orders"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/payments"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/productanalytics"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/productdiscounts"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/products"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/recommendations"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/reviews"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/roles"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/shipments"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/usernotifications"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/users"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/uservouchers"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/vouchers"
	"github.com/gofiber/fiber/v3"
)

type Runtime struct {
	HTTP  *fiber.App
	Close func() error
}

func Bootstrap(ctx context.Context, cfg config.Config, log *slog.Logger) (*Runtime, error) {
	quotes, err := shipments.NewRajaOngkir(cfg.RajaOngkirBaseURL, cfg.RajaOngkirAPIKey, nil)
	if err != nil {
		return nil, fmt.Errorf("shipping quotes: %w", err)
	}
	db, err := infrastructure.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	redisClient, err := infrastructure.NewRedis(ctx, cfg.RedisURL)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("redis: %w", err)
	}
	httpApp := fiber.New(httpConfig(cfg))
	ConfigureCORS(httpApp, cfg)
	store, err := infrastructure.NewMinIO(ctx, cfg.MinIOEndpoint, cfg.MinIOAccessKey, cfg.MinIOSecretKey, cfg.MinIOBucket, cfg.PublicAPIURL, cfg.MinIOUseSSL)
	if err != nil {
		_ = redisClient.Close()
		_ = sqlDB.Close()
		return nil, fmt.Errorf("minio: %w", err)
	}
	ses := sessions.NewService(db, cfg.AccessJWTSecret, cfg.RefreshJWTSecret)
	roles.NewService(db, ses).Routes(httpApp)
	users.NewService(db, ses, store).Routes(httpApp)
	products.NewService(db, store, ses).Routes(httpApp)
	discounts.NewService(db, ses).Routes(httpApp)
	productdiscounts.NewService(db, ses).Routes(httpApp)
	vouchers.NewService(db, ses).Routes(httpApp)
	uservouchers.NewService(db, ses).Routes(httpApp)
	orders.NewService(db, ses).Routes(httpApp)
	payments.NewService().Routes(httpApp)
	shipments.NewService(db, ses, quotes, shipments.NewRedisCostRateLimiter(redisClient, cfg.RajaOngkirRateLimit, time.Minute, cfg.RajaOngkirGlobalRateLimit)).Routes(httpApp)
	reviews.NewService(db, ses).Routes(httpApp)
	recommendations.NewService(db).Routes(httpApp)
	productanalytics.NewService(db, ses).Routes(httpApp)
	notifications.NewService(db, ses).Routes(httpApp)
	usernotifications.NewService(db, ses).Routes(httpApp)
	media.NewService(store).Routes(httpApp)
	carts.NewService(db, ses).Routes(httpApp)
	categories.NewService(db, redisClient, ses).Routes(httpApp)
	inventory.NewService(db, ses).Routes(httpApp)
	addresses.NewService(db, ses).Routes(httpApp)
	ses.Routes(httpApp)
	auth.NewService(db, ses, cfg.Environment == "production").Routes(httpApp)
	health.NewService(sqlDB).Routes(httpApp)
	log.Info("dependencies wired", "modules", []string{"roles", "users", "categories", "products", "carts", "inventory", "addresses", "sessions", "auth", "health", "media", "discounts", "product-discounts", "vouchers", "user-vouchers", "orders", "payments", "shipments", "reviews", "recommendations", "product-analytics", "notifications", "user-notifications"})
	return &Runtime{HTTP: httpApp, Close: func() error {
		err := redisClient.Close()
		if x := sqlDB.Close(); err == nil {
			err = x
		}
		return err
	}}, nil
}

func httpConfig(cfg config.Config) fiber.Config {
	options := fiber.Config{ErrorHandler: httpx.ErrorHandler}
	if len(cfg.TrustedProxyCIDRs) > 0 {
		options.TrustProxy = true
		options.ProxyHeader = fiber.HeaderXForwardedFor
		options.TrustProxyConfig = fiber.TrustProxyConfig{Proxies: cfg.TrustedProxyCIDRs}
	}
	return options
}
