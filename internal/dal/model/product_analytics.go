// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type ProductAnalytics struct {
	ID        string    `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	UserID    string    `gorm:"column:user_id;type:uuid;not null" json:"userId"`
	ProductID string    `gorm:"column:product_id;type:uuid;not null" json:"productId"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	Products  Product   `gorm:"foreignKey:ProductID;references:ID" json:"-"`
	Users     User      `gorm:"foreignKey:UserID;references:ID" json:"-"`
}

func (ProductAnalytics) TableName() string {
	return "product_analytics"
}

func (m *ProductAnalytics) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
