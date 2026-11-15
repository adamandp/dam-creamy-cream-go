// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type ProductDiscount struct {
	ID         string    `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	ProductID  string    `gorm:"column:product_id;type:uuid;not null;uniqueIndex:product_discounts_product_id_key" json:"productId"`
	DiscountID string    `gorm:"column:discount_id;type:uuid;not null" json:"discountId"`
	IsActive   *bool     `gorm:"column:is_active;type:boolean;not null;default:true" json:"isActive"`
	CreatedAt  time.Time `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	Discounts  Discount  `gorm:"foreignKey:DiscountID;references:ID" json:"-"`
	Products   Product   `gorm:"foreignKey:ProductID;references:ID" json:"-"`
}

func (ProductDiscount) TableName() string {
	return "product_discounts"
}

func (m *ProductDiscount) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
