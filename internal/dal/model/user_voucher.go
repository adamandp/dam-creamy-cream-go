// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type UserVoucher struct {
	ID        string            `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	UserID    string            `gorm:"column:user_id;type:uuid;not null" json:"userId"`
	VoucherID string            `gorm:"column:voucher_id;type:uuid;not null" json:"voucherId"`
	Status    UserVoucherStatus `gorm:"column:status;type:\"USER_VOUCHER_STATUS\";not null;default:'CLAIMED'" json:"status"`
	ClaimedAt time.Time         `gorm:"column:claimed_at;type:timestamp(3);not null" json:"claimedAt"`
	ExpiredAt time.Time         `gorm:"column:expired_at;type:timestamp(3);not null" json:"expiredAt"`
	CreatedAt time.Time         `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time         `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	Vouchers  Voucher           `gorm:"foreignKey:VoucherID;references:ID" json:"-"`
	Users     User              `gorm:"foreignKey:UserID;references:ID" json:"-"`
}

func (UserVoucher) TableName() string {
	return "user_vouchers"
}

func (m *UserVoucher) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
