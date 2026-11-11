package roles

import (
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/roles")
	r.Post("/", s.sessions.RequireAdmin, s.create)
	r.Get("/", s.findAll)
	r.Get("/name/:name", s.byName)
	r.Get("/:id", s.byID)
	r.Patch("/:id", s.sessions.RequireAdmin, s.update)
	r.Delete("/:id", s.sessions.RequireAdmin, s.remove)
}
