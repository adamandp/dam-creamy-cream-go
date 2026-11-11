package inventory

import (
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/inventory")
	r.Get("/", s.findAll)
	r.Get("/product/:id", s.byProduct)
	r.Get("/:id", s.byID)
	r.Patch("/:id", s.sessions.RequireAdmin, s.update)
}
