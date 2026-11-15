// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Category struct {
	ID          string    `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	Name        string    `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Description *string   `gorm:"column:description;type:varchar(255)" json:"description"`
	IsActive    *bool     `gorm:"column:is_active;type:boolean;not null;default:true" json:"isActive"`
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	Products    []Product `gorm:"foreignKey:CategoryID;references:ID" json:"-"`
}

func (Category) TableName() string {
	return "category"
}

func (m *Category) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
