// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Discount struct {
	ID               string            `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	Name             string            `gorm:"column:name;type:varchar(50);not null" json:"name"`
	DiscountType     DiscountType      `gorm:"column:discount_type;type:\"DISCOUNT_TYPE\";not null" json:"discountType"`
	Value            *float64          `gorm:"column:value;type:double precision" json:"value"`
	MaxApplied       int               `gorm:"column:max_applied;type:integer;not null" json:"maxApplied"`
	MaxRedeemed      int               `gorm:"column:max_redeemed;type:integer;not null" json:"maxRedeemed"`
	RedeemedCount    int               `gorm:"column:redeemed_count;type:integer;not null;default:0" json:"redeemedCount"`
	Description      *string           `gorm:"column:description;type:varchar(255)" json:"description"`
	ProductID        *string           `gorm:"column:product_id;type:uuid" json:"productId"`
	Quantity         *int              `gorm:"column:quantity;type:integer" json:"quantity"`
	StartDate        time.Time         `gorm:"column:start_date;type:timestamp(3);not null" json:"startDate"`
	EndDate          time.Time         `gorm:"column:end_date;type:timestamp(3);not null" json:"endDate"`
	CreatedAt        time.Time         `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt        time.Time         `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	ProductDiscounts []ProductDiscount `gorm:"foreignKey:DiscountID;references:ID" json:"-"`
	OrderItems       []OrderItem       `gorm:"foreignKey:DiscountID;references:ID" json:"-"`
}

func (Discount) TableName() string {
	return "discounts"
}

func (m *Discount) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
