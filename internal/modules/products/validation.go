package products

import (
	"io"
	"mime/multipart"

	"github.com/gofiber/fiber/v3"
)

func checkPrice(v float64) error {
	if v < 1000 || v > 1000000 {
		return fiber.NewError(400, "price must be between 1000 and 1000000")
	}
	return nil
}

func validatedImage(f *multipart.FileHeader) ([]byte, error) {
	if f == nil {
		return nil, fiber.NewError(400, "file is required")
	}
	if f.Size > 2*1024*1024 {
		return nil, fiber.NewError(400, "file too large")
	}
	mimetype := f.Header.Get("Content-Type")
	switch mimetype {
	case "image/jpg", "image/jpeg", "image/png", "image/webp":
	default:
		return nil, fiber.NewError(400, "Only image files are allowed!")
	}
	src, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()
	bytes, err := io.ReadAll(io.LimitReader(src, 2*1024*1024+1))
	if err != nil {
		return nil, err
	}
	if len(bytes) > 2*1024*1024 {
		return nil, fiber.NewError(400, "file too large")
	}
	return bytes, nil
}
