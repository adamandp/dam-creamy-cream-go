package reviews

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/reviews")
	r.Post("/", s.sessions.RequireActor, s.create)
	r.Get("/", s.findAll)
	r.Get("/product/:id", s.byProduct)
	r.Get("/me", s.sessions.RequireActor, s.byUser)
	r.Get("/:id", s.byID)
	r.Patch("/:id", s.sessions.RequireActor, s.update)
	r.Delete("/:id", s.sessions.RequireActor, s.remove)
}
