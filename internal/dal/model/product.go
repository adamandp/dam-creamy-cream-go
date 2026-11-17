// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Product struct {
	ID                 string              `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	CategoryID         string              `gorm:"column:category_id;type:uuid;not null" json:"categoryId"`
	DiscountID         *string             `gorm:"column:discount_id;type:uuid" json:"discountId"`
	Name               string              `gorm:"column:name;type:varchar(100);not null" json:"name"`
	ImageURL           *string             `gorm:"column:image_url;type:varchar(255)" json:"imageUrl"`
	Description        *string             `gorm:"column:description;type:varchar(255)" json:"description"`
	Price              float64             `gorm:"column:price;type:double precision;not null" json:"price"`
	IsActive           bool                `gorm:"column:is_active;type:boolean;not null" json:"isActive"`
	CreatedAt          time.Time           `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt          time.Time           `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	Categories         Category            `gorm:"foreignKey:CategoryID;references:ID" json:"-"`
	ProductDiscounts   *ProductDiscount    `gorm:"foreignKey:ProductID;references:ID" json:"-"`
	Inventory          *Inventory          `gorm:"foreignKey:ProductID;references:ID" json:"-"`
	CartItems          []CartItem          `gorm:"foreignKey:ProductID;references:ID" json:"-"`
	OrderItems         []OrderItem         `gorm:"foreignKey:ProductID;references:ID" json:"-"`
	Reviews            []Review            `gorm:"foreignKey:ProductID;references:ID" json:"-"`
	Vouchers           []Voucher           `gorm:"foreignKey:ProductID;references:ID" json:"-"`
	ProductAnalytics   []ProductAnalytics  `gorm:"foreignKey:ProductID;references:ID" json:"-"`
	ProductInformation *ProductInformation `gorm:"foreignKey:ProductID;references:ID" json:"-"`
}

func (Product) TableName() string {
	return "products"
}

func (m *Product) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
