package recommendations

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
)

type discount struct {
	Type  string
	Value *float64
}

type recommendation struct {
	Product  domain.Product
	Rate     float64
	Sales    int64
	Discount *discount
}

type recommendationView struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	ImageURL      string   `json:"imageUrl"`
	Description   string   `json:"description"`
	Rate          float64  `json:"rate"`
	Price         float64  `json:"price"`
	DiscountPrice *float64 `json:"discountPrice"`
	DiscountType  *string  `json:"discountType"`
	DiscountValue *float64 `json:"discountValue"`
}
