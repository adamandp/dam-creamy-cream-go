package products

import (
	"dam-creamy-cream-go/internal/dal/model"
	"dam-creamy-cream-go/internal/types"
)

type CreateRequest struct {
	CategoryID  string  `json:"categoryId" validate:"required,uuid"`
	DiscountID  *string `json:"discountId" validate:"omitempty,uuid"`
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	Description *string `json:"description" validate:"omitempty,min=1,max=255"`
	Price       float64 `json:"price" validate:"required,min=1000,max=1000000"`
	IsActive    *bool   `json:"isActive" validate:"required"`
}

type UpdateRequest struct {
	CategoryID  *string  `json:"categoryId" validate:"omitempty,uuid"`
	DiscountID  *string  `json:"discountId" validate:"omitempty,uuid"`
	Name        *string  `json:"name" validate:"omitempty,min=1,max=100"`
	Description *string  `json:"description" validate:"omitempty,min=1,max=255"`
	Price       *float64 `json:"price" validate:"omitempty,min=1000,max=1000000"`
	IsActive    *bool    `json:"isActive"`
	ImageKey    *string  `json:"imageKey" validate:"omitempty,min=1,max=255"`
}

type FindAllRequest struct {
	types.PaginationQuery
}

type FindCatalogRequest struct {
	types.PaginationQuery

	Query *string  `query:"q" validate:"omitempty,min=1,max=100"`
	Cat   *string  `query:"cat" validate:"omitempty,min=1,max=100"`
	Min   *float64 `query:"min" validate:"omitempty,min=0,max=1000000"`
	Max   *float64 `query:"max" validate:"omitempty,min=0,max=1000000"`
	Sort  *string  `query:"sort" validate:"omitempty,oneof=asc desc default"`
}

type PresignImageRequest struct {
	ContentType string `json:"contentType" validate:"required,oneof=image/jpeg image/jpg image/png image/webp"`
	Size        int64  `json:"size" validate:"required,min=1,max=2097152"`
}

type CreateResponse struct {
	ID string `json:"id"`
}

type PresignImageResponse struct {
	ProductID string `json:"productId"`
	UploadURL string `json:"uploadUrl"`
	ImageKey  string `json:"imageKey"`
	ExpiresIn int    `json:"expiresIn"`
}

type UpdateResponse struct {
	ImageURL string `json:"imageUrl"`
}

type CatalogResponse struct {
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
