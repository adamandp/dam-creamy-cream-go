package categories

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func RegisterRoutes(
	router fiber.Router,
	db *gorm.DB,
	redis *redis.Client,
	log *logrus.Entry,
	validate *validator.Validate,
) {
	service := NewService(db, redis, log)
	handler := NewHandler(service, validate)

	group := router.Group("/categories")

	group.Post("", handler.Create)
	group.Get("", handler.FindAll)
	group.Get("/:id", handler.FindByID)
	group.Patch("/:id", handler.Update)
	group.Delete("/:id", handler.Delete)
}
