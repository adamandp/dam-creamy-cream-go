package vouchers

import (
	"time"

	"dam-creamy-cream-go/internal/dal/model"
	"dam-creamy-cream-go/internal/types"
)

type CreateRequest struct {
	Code            string             `json:"code" validate:"required,min=1,max=50"`
	Name            string             `json:"name" validate:"required,min=1,max=50"`
	Description     *string            `json:"description" validate:"omitempty,min=1,max=255"`
	Type            model.VoucherType  `json:"type" validate:"required,oneof=PERCENTAGE FIXED FREE_ITEM"`
	Group           model.VoucherGroup `json:"group" validate:"required,oneof=CHECKOUT SHIPPING"`
	Value           *float64           `json:"value"`
	ProductID       *string            `json:"productId" validate:"omitempty,uuid"`
	Quantity        *int               `json:"quantity" validate:"omitempty,min=-2147483648,max=2147483647"`
	MinimumPurchase *float64           `json:"minimumPurchase" validate:"required"`
	IsActive        *bool              `json:"isActive" validate:"required"`
	StartDate       *time.Time         `json:"startDate" validate:"required"`
	EndDate         *time.Time         `json:"endDate" validate:"required"`
	MaxRedeemed     *int               `json:"maxRedeemed" validate:"required,min=-2147483648,max=2147483647"`
	RedeemedCount   *int               `json:"redeemedCount" validate:"required,min=-2147483648,max=2147483647"`
}

type UpdateRequest struct {
	Code            *string             `json:"code" validate:"omitempty,min=1,max=50"`
	Name            *string             `json:"name" validate:"omitempty,min=1,max=50"`
	Description     *string             `json:"description" validate:"omitempty,min=1,max=255"`
	Type            *model.VoucherType  `json:"type" validate:"omitempty,oneof=PERCENTAGE FIXED FREE_ITEM"`
	Group           *model.VoucherGroup `json:"group" validate:"omitempty,oneof=CHECKOUT SHIPPING"`
	Value           *float64            `json:"value"`
	ProductID       *string             `json:"productId" validate:"omitempty,uuid"`
	Quantity        *int                `json:"quantity" validate:"omitempty,min=-2147483648,max=2147483647"`
	MinimumPurchase *float64            `json:"minimumPurchase"`
	IsActive        *bool               `json:"isActive"`
	StartDate       *time.Time          `json:"startDate"`
	EndDate         *time.Time          `json:"endDate"`
	MaxRedeemed     *int                `json:"maxRedeemed" validate:"omitempty,min=-2147483648,max=2147483647"`
	RedeemedCount   *int                `json:"redeemedCount" validate:"omitempty,min=-2147483648,max=2147483647"`
}

type FindAllRequest struct {
	types.PaginationQuery
}
