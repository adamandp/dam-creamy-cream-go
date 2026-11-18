package recommendations

import "dam-creamy-cream-go/internal/dal/model"

type ProductResponse struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	ImageURL      string              `json:"imageUrl"`
	Description   string              `json:"description"`
	Rate          float64             `json:"rate"`
	Price         float64             `json:"price"`
	DiscountPrice *float64            `json:"discountPrice"`
	DiscountType  *model.DiscountType `json:"discountType"`
	DiscountValue *float64            `json:"discountValue"`
}

type CategoryResponse struct {
	ID         string `json:"id"`
	ImageURL   string `json:"imageUrl"`
	Category   string `json:"category"`
	CategoryID string `json:"categoryId"`
}
