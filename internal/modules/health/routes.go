package health

import (
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	app.Get("/health", s.web)
	app.Get("/health/db", s.postgres)
}
