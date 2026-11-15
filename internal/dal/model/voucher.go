// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Voucher struct {
	ID              string         `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	Code            string         `gorm:"column:code;type:varchar(50);not null" json:"code"`
	Name            string         `gorm:"column:name;type:varchar(50);not null" json:"name"`
	Description     *string        `gorm:"column:description;type:varchar(255)" json:"description"`
	Type            VoucherType    `gorm:"column:type;type:\"VOUCHER_TYPE\";not null" json:"type"`
	Group           VoucherGroup   `gorm:"column:group;type:\"VOUCHER_GROUP\";not null" json:"group"`
	Value           *float64       `gorm:"column:value;type:double precision" json:"value"`
	ProductID       *string        `gorm:"column:product_id;type:uuid" json:"productId"`
	Quantity        *int           `gorm:"column:quantity;type:integer" json:"quantity"`
	MinimumPurchase float64        `gorm:"column:minimum_purchase;type:double precision;not null" json:"minimumPurchase"`
	IsActive        bool           `gorm:"column:is_active;type:boolean;not null" json:"isActive"`
	StartDate       time.Time      `gorm:"column:start_date;type:timestamp(3);not null" json:"startDate"`
	EndDate         time.Time      `gorm:"column:end_date;type:timestamp(3);not null" json:"endDate"`
	MaxRedeemed     int            `gorm:"column:max_redeemed;type:integer;not null" json:"maxRedeemed"`
	RedeemedCount   int            `gorm:"column:redeemed_count;type:integer;not null;default:0" json:"redeemedCount"`
	CreatedAt       time.Time      `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	Products        *Product       `gorm:"foreignKey:ProductID;references:ID" json:"-"`
	UserVouchers    []UserVoucher  `gorm:"foreignKey:VoucherID;references:ID" json:"-"`
	Orders          []OrderVoucher `gorm:"foreignKey:VoucherID;references:ID" json:"-"`
}

func (Voucher) TableName() string {
	return "vouchers"
}

func (m *Voucher) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
