package reviews

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/legacyviews"
	"github.com/gofiber/fiber/v3"
)

func mapReviews(rows []domain.Review) []fiber.Map {
	data := make([]fiber.Map, 0, len(rows))
	for _, row := range rows {
		data = append(data, legacyviews.Review(row))
	}
	return data
}
