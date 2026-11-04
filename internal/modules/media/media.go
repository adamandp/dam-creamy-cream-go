package media

import (
	"context"
	"errors"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
	"github.com/gofiber/fiber/v3"
)

type Service struct {
	store infrastructure.ImageStore
}

func NewService(store infrastructure.ImageStore) *Service {
	return &Service{store: store}
}

func (s *Service) Routes(app fiber.Router) {
	// Same public image visibility as the old Cloudinary URLs, but MinIO stays private.
	app.Get("/media/:scope/:id/:filename", s.read)
}

func (s *Service) read(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	img, err := s.store.Get(ctx, c.Params("scope"), c.Params("id"), c.Params("filename"))
	if err != nil {
		if errors.Is(err, infrastructure.ErrImageNotFound) {
			return fiber.NewError(404, "Image not found")
		}
		return err
	}
	c.Set("Content-Type", img.ContentType)
	c.Set("Cache-Control", "public, max-age=31536000, immutable")
	c.Set("X-Content-Type-Options", "nosniff")
	return c.Send(img.Body)
}
