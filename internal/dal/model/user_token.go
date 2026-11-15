// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type UserToken struct {
	ID          string    `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	UserID      string    `gorm:"column:user_id;type:uuid;not null" json:"userId"`
	Token       string    `gorm:"column:token;type:varchar(255);not null;uniqueIndex:user_tokens_token_key" json:"token"`
	BrowserInfo string    `gorm:"column:browserInfo;type:text;not null" json:"browserInfo"`
	ExpiredAt   time.Time `gorm:"column:expiredAt;type:timestamp(3);not null" json:"expiredAt"`
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	Users       User      `gorm:"foreignKey:UserID;references:ID" json:"-"`
}

func (UserToken) TableName() string {
	return "user_tokens"
}

func (m *UserToken) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
