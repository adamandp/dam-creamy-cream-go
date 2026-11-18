package product

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"dam-creamy-cream-go/internal/dal/model"
)

func buildInformation(
	description string,
	information *model.ProductInformation,
) (string, error) {
	highlights, err := decodeBullets(information.Highlights)
	if err != nil {
		return "", fmt.Errorf("invalid highlights: %w", err)
	}

	servingSuggestions, err := decodeBullets(
		information.ServingSuggestions,
	)
	if err != nil {
		return "", fmt.Errorf("invalid serving suggestions: %w", err)
	}

	var details productDetails

	if err := json.Unmarshal(information.ProductDetails, &details); err != nil {
		return "", fmt.Errorf("invalid product details: %w", err)
	}

	if details.Size == nil ||
		details.Storage == nil ||
		details.ShelfLife == nil {
		return "", errors.New("missing required product details")
	}

	result := fmt.Sprintf(
		`%s

🍦 Highlights
%s

🍨 Serving Suggestions
%s

📦 Product Details
• Size: %s
• Storage: %s
• Shelf life: %s

⚡ Why Choose This Product?
%s`,
		description,
		highlights,
		servingSuggestions,
		*details.Size,
		*details.Storage,
		*details.ShelfLife,
		information.WhyChoose,
	)

	return strings.TrimSpace(result), nil
}

func decodeBullets(data []byte) (string, error) {
	var items []*string

	if err := json.Unmarshal(data, &items); err != nil {
		return "", err
	}

	if items == nil {
		return "", errors.New("expected an array of strings")
	}

	bullets := make([]string, 0, len(items))

	for _, item := range items {
		if item == nil {
			return "", errors.New("expected a string")
		}

		bullets = append(bullets, "• "+*item)
	}

	return strings.Join(bullets, "\n"), nil
}
