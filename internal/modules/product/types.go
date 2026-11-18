package product

import "dam-creamy-cream-go/internal/dal/model"

type FindOneResponse struct {
	ID            string              `json:"id"`
	Name          string              `json:"name"`
	ImageURL      string              `json:"imageUrl"`
	Description   string              `json:"description"`
	Rate          float64             `json:"rate"`
	Category      string              `json:"category"`
	Sales         int64               `json:"sales"`
	Price         float64             `json:"price"`
	DiscountPrice *float64            `json:"discountPrice"`
	DiscountType  *model.DiscountType `json:"discountType"`
	DiscountValue *float64            `json:"discountValue"`
}

type FindInformationResponse struct {
	Information string `json:"information"`
}

type productDetails struct {
	Size      *string `json:"size"`
	Storage   *string `json:"storage"`
	ShelfLife *string `json:"shelfLife"`
}
