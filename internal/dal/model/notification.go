// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Notification struct {
	ID                string             `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	Title             string             `gorm:"column:title;type:varchar(50);not null" json:"title"`
	Message           string             `gorm:"column:message;type:varchar(255);not null" json:"message"`
	CreatedAt         time.Time          `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt         time.Time          `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	UserNotifications []UserNotification `gorm:"foreignKey:NotificationID;references:ID" json:"-"`
}

func (Notification) TableName() string {
	return "notifications"
}

func (m *Notification) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
