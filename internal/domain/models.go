// Package domain defines GORM models for the Creamy Cream PostgreSQL schema.
// The application does not change its database schema at startup.
package domain

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type ID struct {
	ID string `gorm:"column:id;type:uuid;primaryKey" json:"id"`
}

func (m *ID) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}

type Stamp struct {
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

type Role struct {
	ID
	Name string `gorm:"column:name;size:50;unique" json:"name"`
	Stamp
}

func (Role) TableName() string { return "roles" }

type User struct {
	ID
	RoleID      string     `gorm:"column:role_id;type:uuid" json:"roleId"`
	Username    string     `gorm:"column:username" json:"username"`
	Email       *string    `gorm:"column:email" json:"email"`
	PhoneNumber *string    `gorm:"column:phone_number" json:"phoneNumber"`
	Password    string     `gorm:"column:password" json:"-"`
	FullName    string     `gorm:"column:full_name" json:"fullName"`
	BirthDate   *time.Time `gorm:"column:birth_date;type:date" json:"birthDate"`
	ImageURL    *string    `gorm:"column:image_url" json:"imageUrl"`
	Gender      *string    `gorm:"column:gender;type:GENDER_TYPE" json:"gender"`
	IsActive    bool       `gorm:"column:is_active" json:"isActive"`
	Stamp
}

func (User) TableName() string { return "users" }

type UserToken struct {
	ID
	UserID      string    `gorm:"column:user_id;type:uuid" json:"userId"`
	Token       string    `gorm:"column:token" json:"-"`
	BrowserInfo string    `gorm:"column:browserInfo" json:"browserInfo"`
	ExpiredAt   time.Time `gorm:"column:expiredAt" json:"expiredAt"`
	Stamp
}

func (UserToken) TableName() string { return "user_tokens" }

type Address struct {
	ID
	UserID               string `gorm:"column:user_id;type:uuid" json:"userId"`
	RecipientName        string `gorm:"column:recipient_name" json:"recipientName"`
	RecipientPhoneNumber string `gorm:"column:recipient_phone_number" json:"recipientPhoneNumber"`
	Province             string `gorm:"column:province" json:"province"`
	City                 string `gorm:"column:city" json:"city"`
	Subdistrict          string `gorm:"column:subdistrict" json:"subdistrict"`
	Village              string `gorm:"column:village" json:"village"`
	PostalCode           string `gorm:"column:postal_code" json:"postalCode"`
	Address              string `gorm:"column:address" json:"address"`
	IsPrimary            bool   `gorm:"column:is_primary" json:"isPrimary"`
	Stamp
}

func (Address) TableName() string { return "addresses" }

type Category struct {
	ID
	Name        string  `gorm:"column:name" json:"name"`
	Description *string `gorm:"column:description" json:"description"`
	IsActive    bool    `gorm:"column:is_active" json:"isActive"`
	Stamp
}

func (Category) TableName() string { return "category" } // NOT "categories"

type Product struct {
	ID
	CategoryID  string  `gorm:"column:category_id;type:uuid" json:"categoryId"`
	DiscountID  *string `gorm:"column:discount_id;type:uuid" json:"discountId"`
	Name        string  `gorm:"column:name" json:"name"`
	ImageURL    *string `gorm:"column:image_url" json:"imageUrl"`
	Description *string `gorm:"column:description" json:"description"`
	Price       float64 `gorm:"column:price" json:"price"`
	IsActive    bool    `gorm:"column:is_active" json:"isActive"`
	Stamp
}

func (Product) TableName() string { return "products" }

type Inventory struct {
	ID
	ProductID        string `gorm:"column:product_id;type:uuid" json:"productId"`
	Quantity         int    `gorm:"column:quantity" json:"quantity"`
	ReservedQuantity int    `gorm:"column:reserved_quantity" json:"reservedQuantity"`
	IsActive         bool   `gorm:"column:is_active" json:"isActive"`
	Stamp
}

func (Inventory) TableName() string { return "inventory" }

type Cart struct {
	ID
	UserID string `gorm:"column:user_id;type:uuid" json:"userId"`
	Stamp
}

func (Cart) TableName() string { return "carts" }

type CartItem struct {
	ID
	CartID    string `gorm:"column:cart_id;type:uuid" json:"cartId"`
	ProductID string `gorm:"column:product_id;type:uuid" json:"productId"`
	Quantity  int    `gorm:"column:quantity" json:"quantity"`
	Stamp
}

func (CartItem) TableName() string { return "cart_items" }

type Order struct {
	ID
	UserID         string  `gorm:"column:user_id;type:uuid" json:"userId"`
	AddressID      string  `gorm:"column:address_id;type:uuid" json:"addressId"`
	VoucherAmount  float64 `gorm:"column:voucher_amount" json:"voucherAmount"`
	DiscountAmount float64 `gorm:"column:discount_amount" json:"discountAmount"`
	TotalPrice     float64 `gorm:"column:total_price" json:"totalPrice"`
	FinalPrice     float64 `gorm:"column:final_price" json:"finalPrice"`
	Status         string  `gorm:"column:status;type:ORDER_STATUS" json:"status"`
	Stamp
}

func (Order) TableName() string { return "orders" }

type OrderVoucher struct {
	OrderID   string `gorm:"column:order_id;type:uuid;primaryKey" json:"orderId"`
	VoucherID string `gorm:"column:voucher_id;type:uuid;primaryKey" json:"voucherId"`
}

func (OrderVoucher) TableName() string { return "order_vouchers" }

type OrderItem struct {
	ID
	OrderID        string  `gorm:"column:order_id;type:uuid" json:"orderId"`
	ProductID      string  `gorm:"column:product_id;type:uuid" json:"productId"`
	DiscountID     *string `gorm:"column:discount_id;type:uuid" json:"discountId"`
	DiscountAmount float64 `gorm:"column:discount_amount" json:"discountAmount"`
	Price          float64 `gorm:"column:price" json:"price"`
	FinalPrice     float64 `gorm:"column:finalPrice" json:"finalPrice"`
	Quantity       int     `gorm:"column:quantity" json:"quantity"`
	Stamp
}

func (OrderItem) TableName() string { return "order_items" }

type Payment struct {
	ID
	OrderID         string     `gorm:"column:order_id;type:uuid" json:"orderId"`
	Method          string     `gorm:"column:method" json:"method"`
	MethodType      string     `gorm:"column:method_type" json:"methodType"`
	Amount          float64    `gorm:"column:amount" json:"amount"`
	Status          string     `gorm:"column:status;type:PAYMENT_STATUS" json:"status"`
	TransactionCode *string    `gorm:"column:transaction_code" json:"transactionCode"`
	TransactionID   *string    `gorm:"column:transaction_id" json:"transactionId"`
	ExpiryDate      *time.Time `gorm:"column:expiry_date" json:"expiryDate"`
	PaidAt          *time.Time `gorm:"column:paid_at" json:"paidAt"`
	Stamp
}

func (Payment) TableName() string { return "payments" }

type Shipment struct {
	ID
	OrderID        string     `gorm:"column:order_id;type:uuid" json:"orderId"`
	Courier        string     `gorm:"column:courier" json:"courier"`
	Service        string     `gorm:"column:service" json:"service"`
	Cost           float64    `gorm:"column:cost" json:"cost"`
	Estimated      time.Time  `gorm:"column:estimated" json:"estimated"`
	DeliveredAt    *time.Time `gorm:"column:delivered_at" json:"deliveredAt"`
	Description    *string    `gorm:"column:description" json:"description"`
	TrackingNumber *string    `gorm:"column:tracking_number" json:"trackingNumber"`
	Status         string     `gorm:"column:status;type:SHIPMENT_STATUS" json:"status"`
	Stamp
}

func (Shipment) TableName() string { return "shipments" }

type Review struct {
	ID
	UserID    string  `gorm:"column:user_id;type:uuid" json:"userId"`
	ProductID string  `gorm:"column:product_id;type:uuid" json:"productId"`
	Rating    int     `gorm:"column:rating" json:"rating"`
	Comment   *string `gorm:"column:comment" json:"comment"`
	Stamp
}

func (Review) TableName() string { return "reviews" }

type Discount struct {
	ID
	Name          string    `gorm:"column:name" json:"name"`
	DiscountType  string    `gorm:"column:discount_type;type:DISCOUNT_TYPE" json:"discountType"`
	Value         *float64  `gorm:"column:value" json:"value"`
	MaxApplied    int       `gorm:"column:max_applied" json:"maxApplied"`
	MaxRedeemed   int       `gorm:"column:max_redeemed" json:"maxRedeemed"`
	RedeemedCount int       `gorm:"column:redeemed_count" json:"redeemedCount"`
	Description   *string   `gorm:"column:description" json:"description"`
	ProductID     *string   `gorm:"column:product_id;type:uuid" json:"productId"`
	Quantity      *int      `gorm:"column:quantity" json:"quantity"`
	StartDate     time.Time `gorm:"column:start_date" json:"startDate"`
	EndDate       time.Time `gorm:"column:end_date" json:"endDate"`
	Stamp
}

func (Discount) TableName() string { return "discounts" }

type ProductDiscount struct {
	ID
	ProductID  string `gorm:"column:product_id;type:uuid" json:"productId"`
	DiscountID string `gorm:"column:discount_id;type:uuid" json:"discountId"`
	IsActive   bool   `gorm:"column:is_active" json:"isActive"`
	Stamp
}

func (ProductDiscount) TableName() string { return "product_discounts" }

type ProductInformation struct {
	ID
	ProductID          string `gorm:"column:product_id;type:uuid" json:"productId"`
	Highlights         []byte `gorm:"column:highlights;type:jsonb" json:"highlights"`
	ServingSuggestions []byte `gorm:"column:serving_suggestions;type:jsonb" json:"servingSuggestions"`
	ProductDetails     []byte `gorm:"column:product_details;type:jsonb" json:"productDetails"`
	WhyChoose          string `gorm:"column:why_choose" json:"whyChoose"`
	Stamp
}

func (ProductInformation) TableName() string { return "product_information" }

type Voucher struct {
	ID
	Code            string    `gorm:"column:code" json:"code"`
	Name            string    `gorm:"column:name" json:"name"`
	Description     *string   `gorm:"column:description" json:"description"`
	Type            string    `gorm:"column:type;type:VOUCHER_TYPE" json:"type"`
	Group           string    `gorm:"column:group;type:VOUCHER_GROUP" json:"group"`
	Value           *float64  `gorm:"column:value" json:"value"`
	ProductID       *string   `gorm:"column:product_id;type:uuid" json:"productId"`
	Quantity        *int      `gorm:"column:quantity" json:"quantity"`
	MinimumPurchase float64   `gorm:"column:minimum_purchase" json:"minimumPurchase"`
	IsActive        bool      `gorm:"column:is_active" json:"isActive"`
	StartDate       time.Time `gorm:"column:start_date" json:"startDate"`
	EndDate         time.Time `gorm:"column:end_date" json:"endDate"`
	MaxRedeemed     int       `gorm:"column:max_redeemed" json:"maxRedeemed"`
	RedeemedCount   int       `gorm:"column:redeemed_count" json:"redeemedCount"`
	Stamp
}

func (Voucher) TableName() string { return "vouchers" }

type UserVoucher struct {
	ID
	UserID    string    `gorm:"column:user_id;type:uuid" json:"userId"`
	VoucherID string    `gorm:"column:voucher_id;type:uuid" json:"voucherId"`
	Status    string    `gorm:"column:status;type:USER_VOUCHER_STATUS" json:"status"`
	ClaimedAt time.Time `gorm:"column:claimed_at" json:"claimedAt"`
	ExpiredAt time.Time `gorm:"column:expired_at" json:"expiredAt"`
	Stamp
}

func (UserVoucher) TableName() string { return "user_vouchers" }

type ProductAnalytics struct {
	ID
	UserID    string    `gorm:"column:user_id;type:uuid" json:"userId"`
	ProductID string    `gorm:"column:product_id;type:uuid" json:"productId"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
}

func (ProductAnalytics) TableName() string { return "product_analytics" }

type Notification struct {
	ID
	Title   string `gorm:"column:title" json:"title"`
	Message string `gorm:"column:message" json:"message"`
	Stamp
}

func (Notification) TableName() string { return "notifications" }

type UserNotification struct {
	ID
	UserID         string `gorm:"column:user_id;type:uuid" json:"userId"`
	NotificationID string `gorm:"column:notification_id;type:uuid" json:"notificationId"`
	IsRead         bool   `gorm:"column:is_read" json:"isRead"`
	Stamp
}

func (UserNotification) TableName() string { return "user_notifications" }
