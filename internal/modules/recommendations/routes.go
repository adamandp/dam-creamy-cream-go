package recommendations

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

	group := router.Group("/recommendations")

	group.Get("/classic", handler.GetClassic)
	group.Get("/categories", handler.GetCategories)
	group.Get("/offers", handler.GetOffers)
	group.Get("/related/:id", handler.GetRelated)
}
