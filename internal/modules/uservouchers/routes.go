package uservouchers

import (
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/user-vouchers")
	r.Use(s.sessions.RequireActor)
	r.Post("/", s.create)
	r.Get("/", s.findAll)
	r.Get("/user", s.byUser)
	r.Get("/voucher/:id", s.byVoucher)
	r.Patch("/:id", s.use)
	r.Get("/:id", s.byID)
}
