// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Review struct {
	ID        string    `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	UserID    string    `gorm:"column:user_id;type:uuid;not null" json:"userId"`
	ProductID string    `gorm:"column:product_id;type:uuid;not null" json:"productId"`
	Rating    int       `gorm:"column:rating;type:integer;not null" json:"rating"`
	Comment   *string   `gorm:"column:comment;type:varchar(255)" json:"comment"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	Users     User      `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Products  Product   `gorm:"foreignKey:ProductID;references:ID" json:"-"`
}

func (Review) TableName() string {
	return "reviews"
}

func (m *Review) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
