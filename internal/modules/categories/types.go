package categories

import "dam-creamy-cream-go/internal/types"

type CreateRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,min=1,max=255"`
	IsActive    *bool   `json:"isActive" validate:"required"`
}

type UpdateRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,min=1,max=255"`
	IsActive    *bool   `json:"isActive"`
}

type FindAllRequest struct {
	types.PaginationQuery
}
