// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type CartItem struct {
	ID        string    `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	CartID    string    `gorm:"column:cart_id;type:uuid;not null;uniqueIndex:cart_items_cart_id_product_id_key;priority:1" json:"cartId"`
	ProductID string    `gorm:"column:product_id;type:uuid;not null;uniqueIndex:cart_items_cart_id_product_id_key;priority:2" json:"productId"`
	Quantity  int       `gorm:"column:quantity;type:integer;not null" json:"quantity"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	Carts     Cart      `gorm:"foreignKey:CartID;references:ID" json:"-"`
	Products  Product   `gorm:"foreignKey:ProductID;references:ID" json:"-"`
}

func (CartItem) TableName() string {
	return "cart_items"
}

func (m *CartItem) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
