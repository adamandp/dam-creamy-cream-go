package carts

import (
	"time"
)

type Input struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

type CartRow struct {
	ProductID     string     `gorm:"column:product_id"`
	Quantity      int        `gorm:"column:quantity"`
	Name          string     `gorm:"column:name"`
	ImageURL      *string    `gorm:"column:image_url"`
	Category      string     `gorm:"column:category"`
	Price         float64    `gorm:"column:price"`
	DiscountType  *string    `gorm:"column:discount_type"`
	DiscountValue *float64   `gorm:"column:discount_value"`
	StartDate     *time.Time `gorm:"column:start_date"`
	EndDate       *time.Time `gorm:"column:end_date"`
}

type CartResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	ImageURL      *string  `json:"imageUrl"`
	Category      string   `json:"category"`
	Price         float64  `json:"price"`
	DiscountPrice *float64 `json:"discountPrice"`
	Qty           int      `json:"qty"`
}
