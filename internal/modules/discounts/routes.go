package discounts

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func RegisterRoutes(
	router fiber.Router,
	db *gorm.DB,
	log *logrus.Entry,
	validate *validator.Validate,
) *Service {
	service := NewService(db, log)
	handler := NewHandler(service, validate)

	group := router.Group("/discounts")

	group.Post("", handler.Create)
	group.Get("", handler.FindAll)
	group.Get("/:id", handler.FindByID)
	group.Patch("/:id", handler.Update)
	group.Delete("/:id", handler.Delete)

	return service
}
