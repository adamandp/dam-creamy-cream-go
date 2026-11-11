package media

import (
	"github.com/gofiber/fiber/v3"
)

func (s *Service) Routes(app fiber.Router) {
	// Same public image visibility as the old Cloudinary URLs, but MinIO stays private.
	app.Get("/media/:scope/:id/:filename", s.read)
}
