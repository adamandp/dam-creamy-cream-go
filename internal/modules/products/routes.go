package products

import (
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	p := app.Group("/products")
	p.Post("/", s.sessions.RequireAdmin, s.create)
	p.Get("/", s.findAll)
	p.Get("/catalog", s.catalog)
	p.Patch("/:id", s.sessions.RequireAdmin, s.update)
	p.Patch("/:id/image", s.sessions.RequireAdmin, s.updateImage)
	p.Delete("/:id", s.sessions.RequireAdmin, s.remove)
	details := app.Group("/product")
	details.Get("/information/:id", s.information)
	details.Get("/:id", s.detail)
}
