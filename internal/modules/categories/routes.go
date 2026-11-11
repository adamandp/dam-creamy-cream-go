package categories

import (
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/categories")
	r.Post("/", s.sessions.RequireAdmin, s.create)
	r.Get("/", s.findAll)
	r.Get("/:id", s.findOne)
	r.Patch("/:id", s.sessions.RequireAdmin, s.update)
	r.Delete("/:id", s.sessions.RequireAdmin, s.remove)
}
