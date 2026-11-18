package products

import (
	"dam-creamy-cream-go/internal/modules/categories"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func RegisterRoutes(
	router fiber.Router,
	db *gorm.DB,
	categoriesService *categories.Service,
	minioClient *minio.Client,
	minioBucket string,
	log *logrus.Entry,
	validate *validator.Validate,
) *Service {
	service := NewService(
		db,
		categoriesService,
		minioClient,
		minioBucket,
		log,
	)

	handler := NewHandler(service, validate)

	group := router.Group("/products")

	group.Post("", handler.Create)
	group.Get("", handler.FindAll)
	group.Get("/catalog", handler.FindCatalog)
	group.Post("/:id/image/presign", handler.PresignImage)
	group.Patch("/:id", handler.Update)
	group.Delete("/:id", handler.Delete)

	return service
}
