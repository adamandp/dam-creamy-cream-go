package productanalytics

import (
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/product-analytics")
	r.Post("/", s.sessions.RequireActor, s.create)
	r.Get("/product/:id/count", s.productCount)
	r.Get("/product/:id", s.byProduct)
	r.Get("/user", s.sessions.RequireActor, s.byUser)
}
