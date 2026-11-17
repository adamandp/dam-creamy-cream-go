// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Order struct {
	ID             string         `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	UserID         string         `gorm:"column:user_id;type:uuid;not null" json:"userId"`
	AddressID      string         `gorm:"column:address_id;type:uuid;not null" json:"addressId"`
	VoucherAmount  float64        `gorm:"column:voucher_amount;type:double precision;not null;default:0" json:"voucherAmount"`
	DiscountAmount float64        `gorm:"column:discount_amount;type:double precision;not null;default:0" json:"discountAmount"`
	TotalPrice     float64        `gorm:"column:total_price;type:double precision;not null" json:"totalPrice"`
	FinalPrice     float64        `gorm:"column:final_price;type:double precision;not null" json:"finalPrice"`
	Status         OrderStatus    `gorm:"column:status;type:\"ORDER_STATUS\";not null;default:'PENDING'" json:"status"`
	CreatedAt      time.Time      `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	Addresses      Address        `gorm:"foreignKey:AddressID;references:ID" json:"-"`
	Users          User           `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Vouchers       []OrderVoucher `gorm:"foreignKey:OrderID;references:ID" json:"-"`
	OrderItems     []OrderItem    `gorm:"foreignKey:OrderID;references:ID" json:"-"`
	Payments       []Payment      `gorm:"foreignKey:OrderID;references:ID" json:"-"`
	Shipments      []Shipment     `gorm:"foreignKey:OrderID;references:ID" json:"-"`
}

func (Order) TableName() string {
	return "orders"
}

func (m *Order) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
