package health

import (
	"github.com/gofiber/fiber/v3"
)

// RegisterRoutes wires the health module and registers its public probes.
func RegisterRoutes(router fiber.Router, params Params) {
	service := NewService(params)
	handler := NewHandler(service)

	router.Get("/health/go", handler.CheckGo)
	router.Get("/health/db", handler.CheckDB)
	router.Get("/health/redis", handler.CheckRedis)
	router.Get("/health/minio", handler.CheckMinIO)
}
