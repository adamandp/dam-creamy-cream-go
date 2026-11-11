package usernotifications

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/user-notifications")
	r.Post("/", s.sessions.RequireAdmin, s.create)
	r.Get("/", s.sessions.RequireAdmin, s.findAll)
	r.Get("/me", s.sessions.RequireActor, s.byUser)
	r.Get("/count/me", s.sessions.RequireActor, s.userCount)
	r.Get("/notification/:id/count", s.sessions.RequireAdmin, s.notificationCount)
	r.Get("/notification/:id", s.sessions.RequireAdmin, s.byNotification)
	r.Get("/:id", s.sessions.RequireActor, s.byID)
	r.Patch("/:id", s.sessions.RequireActor, s.updateRead)
}
