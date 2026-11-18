package inventory

import "dam-creamy-cream-go/internal/types"

type FindAllRequest struct {
	types.PaginationQuery
}

type UpdateRequest struct {
	Quantity         *int  `json:"quantity" validate:"omitempty,min=0,max=2147483647"`
	ReservedQuantity *int  `json:"reservedQuantity" validate:"omitempty,min=0,max=2147483647"`
	IsActive         *bool `json:"isActive"`
}
