package notifications

import (
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/notifications")
	r.Post("/", s.sessions.RequireAdmin, s.create)
	r.Get("/", s.findAll)
	r.Get("/:id", s.byID)
	r.Patch("/:id", s.sessions.RequireAdmin, s.update)
	r.Delete("/:id", s.sessions.RequireAdmin, s.remove)
}
