// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Shipment struct {
	ID             string         `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	OrderID        string         `gorm:"column:order_id;type:uuid;not null" json:"orderId"`
	Courier        string         `gorm:"column:courier;type:varchar(50);not null" json:"courier"`
	Service        string         `gorm:"column:service;type:varchar(50);not null" json:"service"`
	Cost           float64        `gorm:"column:cost;type:double precision;not null" json:"cost"`
	Estimated      time.Time      `gorm:"column:estimated;type:timestamp(3);not null" json:"estimated"`
	DeliveredAt    *time.Time     `gorm:"column:delivered_at;type:timestamp(3)" json:"deliveredAt"`
	Description    *string        `gorm:"column:description;type:varchar(255)" json:"description"`
	TrackingNumber *string        `gorm:"column:tracking_number;type:varchar(50)" json:"trackingNumber"`
	Status         ShipmentStatus `gorm:"column:status;type:\"SHIPMENT_STATUS\";not null;default:'PENDING'" json:"status"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	CreatedAt      time.Time      `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	Orders         Order          `gorm:"foreignKey:OrderID;references:ID" json:"-"`
}

func (Shipment) TableName() string {
	return "shipments"
}

func (m *Shipment) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
