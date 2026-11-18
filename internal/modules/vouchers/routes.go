package vouchers

import (
	"dam-creamy-cream-go/internal/modules/products"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func RegisterRoutes(
	router fiber.Router,
	db *gorm.DB,
	productsService *products.Service,
	log *logrus.Entry,
	validate *validator.Validate,
) *Service {
	service := NewService(db, productsService, log)
	handler := NewHandler(service, validate)

	group := router.Group("/vouchers")

	group.Post("", handler.Create)
	group.Get("", handler.FindAll)
	group.Get("/:id", handler.FindByID)
	group.Patch("/:id", handler.Update)
	group.Delete("/:id", handler.Delete)

	return service
}
