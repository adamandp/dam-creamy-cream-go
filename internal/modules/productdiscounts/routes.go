package productdiscounts

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/product-discounts")
	r.Post("/", s.sessions.RequireAdmin, s.create)
	r.Get("/", s.findAll)
	r.Get("/product/:productId", s.byProductID)
	r.Get("/:id", s.byID)
	r.Delete("/:id", s.sessions.RequireAdmin, s.remove)
}
