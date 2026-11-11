package addresses

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/addresses")
	r.Post("/", s.sessions.RequireActor, s.create)
	r.Get("/", s.sessions.RequireAdmin, s.findAll)
	r.Get("/user", s.findByUser)
	r.Get("/:id", s.sessions.RequireActor, s.findOne)
	r.Patch("/:id", s.sessions.RequireActor, s.update)
	r.Delete("/:id", s.sessions.RequireActor, s.remove)
}
