package auth

import (
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/auth")
	r.Post("/dummy-customer-login", s.dummyLogin)
	r.Post("/logout", s.logout)
	r.Post("/logout-all", s.logoutAll)
}
