// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Inventory struct {
	ID               string    `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	ProductID        string    `gorm:"column:product_id;type:uuid;not null;uniqueIndex:inventory_product_id_key" json:"productId"`
	Quantity         int       `gorm:"column:quantity;type:integer;not null" json:"quantity"`
	ReservedQuantity int       `gorm:"column:reserved_quantity;type:integer;not null;default:0" json:"reservedQuantity"`
	IsActive         bool      `gorm:"column:is_active;type:boolean;not null" json:"isActive"`
	CreatedAt        time.Time `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt        time.Time `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	Products         Product   `gorm:"foreignKey:ProductID;references:ID" json:"-"`
}

func (Inventory) TableName() string {
	return "inventory"
}

func (m *Inventory) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
