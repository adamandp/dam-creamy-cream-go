package infrastructure

import (
	"dam-creamy-cream-go/internal/modules/categories"
	"dam-creamy-cream-go/internal/modules/discounts"
	"dam-creamy-cream-go/internal/modules/health"
	"dam-creamy-cream-go/internal/modules/inventory"
	"dam-creamy-cream-go/internal/modules/product"
	"dam-creamy-cream-go/internal/modules/productdiscounts"
	"dam-creamy-cream-go/internal/modules/products"
	"dam-creamy-cream-go/internal/modules/recommendations"
	"dam-creamy-cream-go/internal/modules/vouchers"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func registerModules(
	router fiber.Router,
	db *gorm.DB,
	redisClient *redis.Client,
	minioClient *minio.Client,
	minioBucket string,
	log *logrus.Entry,
	validate *validator.Validate,
) {
	health.RegisterRoutes(router, health.Params{
		Database:    db,
		Redis:       redisClient,
		MinIO:       minioClient,
		MinIOBucket: minioBucket,
	})

	categoriesService := categories.RegisterRoutes(
		router,
		db,
		redisClient,
		log,
		validate,
	)

	productsService := products.RegisterRoutes(
		router,
		db,
		categoriesService,
		minioClient,
		minioBucket,
		log,
		validate,
	)

	discountsService := discounts.RegisterRoutes(
		router,
		db,
		log,
		validate,
	)

	productdiscounts.RegisterRoutes(
		router,
		db,
		productsService,
		discountsService,
		log,
		validate,
	)

	inventory.RegisterRoutes(
		router,
		db,
		log,
		validate,
	)

	product.RegisterRoutes(
		router,
		db,
		log,
		validate,
	)

	recommendations.RegisterRoutes(
		router,
		db,
		log,
		validate,
	)

	vouchers.RegisterRoutes(
		router,
		db,
		productsService,
		log,
		validate,
	)
}
