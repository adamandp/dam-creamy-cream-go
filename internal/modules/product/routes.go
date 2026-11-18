package product

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
) {
	service := NewService(db, log)
	handler := NewHandler(service, validate)

	group := router.Group("/product")

	group.Get("/information/:id", handler.FindInformation)
	group.Get("/:id", handler.FindOne)
}
