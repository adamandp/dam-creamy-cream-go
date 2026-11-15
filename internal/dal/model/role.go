// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Role struct {
	ID        string    `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	Name      string    `gorm:"column:name;type:varchar(50);not null;uniqueIndex:roles_name_key" json:"name"`
	CreatedAt time.Time `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	Users     []User    `gorm:"foreignKey:RoleID;references:ID" json:"-"`
}

func (Role) TableName() string {
	return "roles"
}

func (m *Role) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
