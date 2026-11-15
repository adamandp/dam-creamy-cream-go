// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Payment struct {
	ID              string        `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	OrderID         string        `gorm:"column:order_id;type:uuid;not null" json:"orderId"`
	Method          string        `gorm:"column:method;type:varchar(50);not null" json:"method"`
	MethodType      string        `gorm:"column:method_type;type:varchar(50);not null" json:"methodType"`
	Amount          float64       `gorm:"column:amount;type:double precision;not null" json:"amount"`
	Status          PaymentStatus `gorm:"column:status;type:\"PAYMENT_STATUS\";not null;default:'PENDING'" json:"status"`
	TransactionCode *string       `gorm:"column:transaction_code;type:varchar(255)" json:"transactionCode"`
	TransactionID   *string       `gorm:"column:transaction_id;type:varchar(255);uniqueIndex:payments_transaction_id_key" json:"transactionId"`
	ExpiryDate      *time.Time    `gorm:"column:expiry_date;type:timestamp(3)" json:"expiryDate"`
	PaidAt          *time.Time    `gorm:"column:paid_at;type:timestamp(3)" json:"paidAt"`
	UpdatedAt       time.Time     `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	CreatedAt       time.Time     `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	Orders          Order         `gorm:"foreignKey:OrderID;references:ID" json:"-"`
}

func (Payment) TableName() string {
	return "payments"
}

func (m *Payment) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
