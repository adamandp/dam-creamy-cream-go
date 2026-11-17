// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Address struct {
	ID                   string    `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	UserID               string    `gorm:"column:user_id;type:uuid;not null" json:"userId"`
	RecipientName        string    `gorm:"column:recipient_name;type:varchar(100);not null" json:"recipientName"`
	RecipientPhoneNumber string    `gorm:"column:recipient_phone_number;type:varchar(100);not null" json:"recipientPhoneNumber"`
	Province             string    `gorm:"column:province;type:varchar(100);not null" json:"province"`
	City                 string    `gorm:"column:city;type:varchar(100);not null" json:"city"`
	Subdistrict          string    `gorm:"column:subdistrict;type:varchar(100);not null" json:"subdistrict"`
	Village              string    `gorm:"column:village;type:varchar(100);not null" json:"village"`
	PostalCode           string    `gorm:"column:postal_code;type:varchar(50);not null" json:"postalCode"`
	Address              string    `gorm:"column:address;type:varchar(255);not null" json:"address"`
	IsPrimary            *bool     `gorm:"column:is_primary;type:boolean;not null;default:false" json:"isPrimary"`
	CreatedAt            time.Time `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt            time.Time `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	Users                User      `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Orders               []Order   `gorm:"foreignKey:AddressID;references:ID" json:"-"`
}

func (Address) TableName() string {
	return "addresses"
}

func (m *Address) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
