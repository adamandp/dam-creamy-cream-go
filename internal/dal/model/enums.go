package model

// These types correspond to the existing PostgreSQL enum types.

// GenderType maps to the PostgreSQL enum GENDER_TYPE.
type GenderType string

const (
	GenderTypeMale   GenderType = "MALE"
	GenderTypeFemale GenderType = "FEMALE"
	GenderTypeOther  GenderType = "OTHER"
)

// OrderStatus maps to the PostgreSQL enum ORDER_STATUS.
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusPaid      OrderStatus = "PAID"
	OrderStatusShipped   OrderStatus = "SHIPPED"
	OrderStatusDelivered OrderStatus = "DELIVERED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

// PaymentStatus maps to the PostgreSQL enum PAYMENT_STATUS.
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "PENDING"
	PaymentStatusPaid      PaymentStatus = "PAID"
	PaymentStatusFailed    PaymentStatus = "FAILED"
	PaymentStatusCancelled PaymentStatus = "CANCELLED"
)

// ShipmentStatus maps to the PostgreSQL enum SHIPMENT_STATUS.
type ShipmentStatus string

const (
	ShipmentStatusPending   ShipmentStatus = "PENDING"
	ShipmentStatusShipped   ShipmentStatus = "SHIPPED"
	ShipmentStatusDelivered ShipmentStatus = "DELIVERED"
	ShipmentStatusCancelled ShipmentStatus = "CANCELLED"
)

// DiscountType maps to the PostgreSQL enum DISCOUNT_TYPE.
type DiscountType string

const (
	DiscountTypePercentage DiscountType = "PERCENTAGE"
	DiscountTypeFixed      DiscountType = "FIXED"
	DiscountTypePrice      DiscountType = "PRICE"
	DiscountTypeFreeItem   DiscountType = "FREE_ITEM"
)

// VoucherType maps to the PostgreSQL enum VOUCHER_TYPE.
type VoucherType string

const (
	VoucherTypePercentage VoucherType = "PERCENTAGE"
	VoucherTypeFixed      VoucherType = "FIXED"
	VoucherTypeFreeItem   VoucherType = "FREE_ITEM"
)

// VoucherGroup maps to the PostgreSQL enum VOUCHER_GROUP.
type VoucherGroup string

const (
	VoucherGroupCheckout VoucherGroup = "CHECKOUT"
	VoucherGroupShipping VoucherGroup = "SHIPPING"
)

// UserVoucherStatus maps to the PostgreSQL enum USER_VOUCHER_STATUS.
type UserVoucherStatus string

const (
	UserVoucherStatusUsed    UserVoucherStatus = "USED"
	UserVoucherStatusClaimed UserVoucherStatus = "CLAIMED"
)
