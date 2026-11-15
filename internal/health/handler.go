// Package health exposes the application's HTTP health checks.
package health

import (
	"dam-creamy-cream-go/internal/types"

	"github.com/gofiber/fiber/v3"
)

// Handler translates health results into HTTP responses.
type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// CheckGo handles the application's liveness probe.
func (h *Handler) CheckGo(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Status(fiber.StatusOK).JSON(types.WebResponse[Result]{Message: "Application is healthy", Data: h.service.CheckGo()})
}

// CheckDB handles the database readiness probe.
func (h *Handler) CheckDB(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	result := h.service.CheckDB(c.Context())
	if result.Status != "ok" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(types.WebResponse[Result]{
			Message: "Database health check failed",
			Errors:  "Database unavailable",
			Data:    result,
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.WebResponse[Result]{Message: "Database is healthy", Data: result})
}

// CheckRedis handles the Redis readiness probe.
func (h *Handler) CheckRedis(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	result := h.service.CheckRedis(c.Context())
	if result.Status != "ok" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(types.WebResponse[Result]{
			Message: "Redis health check failed",
			Errors:  "Redis unavailable",
			Data:    result,
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.WebResponse[Result]{Message: "Redis is healthy", Data: result})
}

// CheckMinIO handles the MinIO readiness probe.
func (h *Handler) CheckMinIO(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")

	result := h.service.CheckMinIO(c.Context())

	if result.Status != "ok" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(types.WebResponse[Result]{
			Message: "MinIO health check failed",
			Errors:  "MinIO unavailable",
			Data:    result,
		})
	}

	return c.Status(fiber.StatusOK).JSON(types.WebResponse[Result]{Message: "MinIO is healthy", Data: result})
}
