package discounts

import (
	"time"

	"dam-creamy-cream-go/internal/dal/model"
	"dam-creamy-cream-go/internal/types"
)

type CreateRequest struct {
	Name          string             `json:"name" validate:"required,min=1,max=50"`
	DiscountType  model.DiscountType `json:"discountType" validate:"required,oneof=PERCENTAGE FIXED PRICE FREE_ITEM"`
	Value         *float64           `json:"value"`
	MaxApplied    *int               `json:"maxApplied" validate:"required,min=-2147483648,max=2147483647"`
	MaxRedeemed   *int               `json:"maxRedeemed" validate:"required,min=-2147483648,max=2147483647"`
	RedeemedCount *int               `json:"redeemedCount" validate:"required,min=-2147483648,max=2147483647"`
	Description   *string            `json:"description" validate:"omitempty,min=1,max=255"`
	ProductID     *string            `json:"productId" validate:"omitempty,uuid"`
	Quantity      *int               `json:"quantity" validate:"omitempty,min=-2147483648,max=2147483647"`
	StartDate     *time.Time         `json:"startDate" validate:"required"`
	EndDate       *time.Time         `json:"endDate" validate:"required"`
}

type UpdateRequest struct {
	Name          *string             `json:"name" validate:"omitempty,min=1,max=50"`
	DiscountType  *model.DiscountType `json:"discountType" validate:"omitempty,oneof=PERCENTAGE FIXED PRICE FREE_ITEM"`
	Value         *float64            `json:"value"`
	MaxApplied    *int                `json:"maxApplied" validate:"omitempty,min=-2147483648,max=2147483647"`
	MaxRedeemed   *int                `json:"maxRedeemed" validate:"omitempty,min=-2147483648,max=2147483647"`
	RedeemedCount *int                `json:"redeemedCount" validate:"omitempty,min=-2147483648,max=2147483647"`
	Description   *string             `json:"description" validate:"omitempty,min=1,max=255"`
	ProductID     *string             `json:"productId" validate:"omitempty,uuid"`
	Quantity      *int                `json:"quantity" validate:"omitempty,min=-2147483648,max=2147483647"`
	StartDate     *time.Time          `json:"startDate"`
	EndDate       *time.Time          `json:"endDate"`
}

type FindAllRequest struct {
	types.PaginationQuery
}
