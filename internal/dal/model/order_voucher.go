// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

type OrderVoucher struct {
	OrderID   string  `gorm:"column:order_id;type:uuid;primaryKey;not null" json:"orderId"`
	VoucherID string  `gorm:"column:voucher_id;type:uuid;primaryKey;not null" json:"voucherId"`
	Order     Order   `gorm:"foreignKey:OrderID;references:ID" json:"-"`
	Voucher   Voucher `gorm:"foreignKey:VoucherID;references:ID" json:"-"`
}

func (OrderVoucher) TableName() string {
	return "order_vouchers"
}
