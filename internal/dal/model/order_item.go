// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type OrderItem struct {
	ID             string    `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	OrderID        string    `gorm:"column:order_id;type:uuid;not null" json:"orderId"`
	ProductID      string    `gorm:"column:product_id;type:uuid;not null" json:"productId"`
	DiscountID     *string   `gorm:"column:discount_id;type:uuid" json:"discountId"`
	DiscountAmount float64   `gorm:"column:discount_amount;type:double precision;not null;default:0" json:"discountAmount"`
	Price          float64   `gorm:"column:price;type:double precision;not null" json:"price"`
	FinalPrice     float64   `gorm:"column:finalPrice;type:double precision;not null" json:"finalPrice"`
	Quantity       int       `gorm:"column:quantity;type:integer;not null" json:"quantity"`
	CreatedAt      time.Time `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt      time.Time `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	Orders         Order     `gorm:"foreignKey:OrderID;references:ID" json:"-"`
	Products       Product   `gorm:"foreignKey:ProductID;references:ID" json:"-"`
	Discounts      *Discount `gorm:"foreignKey:DiscountID;references:ID" json:"-"`
}

func (OrderItem) TableName() string {
	return "order_items"
}

func (m *OrderItem) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
