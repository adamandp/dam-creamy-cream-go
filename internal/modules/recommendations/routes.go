package recommendations

import (
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/recommendations")
	r.Get("/classic", s.classic)
	r.Get("/categories", s.categories)
	r.Get("/offers", s.offers)
	r.Get("/related/:id", s.related)
}
