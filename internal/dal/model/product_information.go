// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

type ProductInformation struct {
	ID                 string         `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	ProductID          string         `gorm:"column:product_id;type:uuid;not null;uniqueIndex:product_information_product_id_key" json:"productId"`
	Highlights         datatypes.JSON `gorm:"column:highlights;type:jsonb;not null" json:"highlights"`
	ServingSuggestions datatypes.JSON `gorm:"column:serving_suggestions;type:jsonb;not null" json:"servingSuggestions"`
	ProductDetails     datatypes.JSON `gorm:"column:product_details;type:jsonb;not null" json:"productDetails"`
	WhyChoose          string         `gorm:"column:why_choose;type:text;not null" json:"whyChoose"`
	CreatedAt          time.Time      `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	Products           Product        `gorm:"foreignKey:ProductID;references:ID" json:"-"`
}

func (ProductInformation) TableName() string {
	return "product_information"
}

func (m *ProductInformation) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
