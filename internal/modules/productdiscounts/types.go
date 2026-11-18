package productdiscounts

import "dam-creamy-cream-go/internal/types"

type CreateRequest struct {
	ProductID  string `json:"productId" validate:"required,uuid"`
	DiscountID string `json:"discountId" validate:"required,uuid"`
}

type FindAllRequest struct {
	types.PaginationQuery
}
