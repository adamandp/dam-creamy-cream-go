package carts

import (
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/carts")
	r.Post("/user/add", s.add)
	r.Post("/user/remove", s.remove)
	r.Get("/user", s.byUser)
}
