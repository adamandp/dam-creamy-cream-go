package users

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/users")
	r.Get("/", s.sessions.RequireAdmin, s.findAll)
	r.Get("/profile", s.sessions.RequireActor, s.profile)
	r.Get("/:id", s.sessions.RequireSelfOrAdmin, s.findOne)
	r.Patch("/:id", s.sessions.RequireSelfOrAdmin, s.update)
	r.Put("/:id/password", s.sessions.RequireSelf, s.updatePassword)
	r.Put("/:id/image", s.sessions.RequireSelfOrAdmin, s.updateImage)
	r.Delete("/:id", s.sessions.RequireAdmin, s.remove)
}
