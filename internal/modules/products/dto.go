package products

type CreateInput struct {
	CategoryID  string  `json:"categoryId"`
	DiscountID  *string `json:"discountId"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Price       float64 `json:"price"`
	IsActive    *bool   `json:"isActive"`
}

type UpdateInput struct {
	CategoryID  *string  `json:"categoryId"`
	DiscountID  *string  `json:"discountId"`
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Price       *float64 `json:"price"`
	IsActive    *bool    `json:"isActive"`
}

type CatalogItem struct {
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

type DetailItem struct {
	CatalogItem
	Category string `json:"category"`
	Sales    int64  `json:"sales"`
}

type ReviewRating struct {
	ProductID string  `gorm:"column:product_id"`
	Average   float64 `gorm:"column:average"`
}

type ActiveDiscount struct {
	ProductID string   `gorm:"column:product_id"`
	Type      string   `gorm:"column:discount_type"`
	Value     *float64 `gorm:"column:value"`
}
