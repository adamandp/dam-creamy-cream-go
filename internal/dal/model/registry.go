package model

// All returns the manually defined models for code generation and migration.
func All() []any {
	return []any{
		Role{},
		User{},
		UserToken{},
		Address{},
		Product{},
		Category{},
		Inventory{},
		Cart{},
		CartItem{},
		Order{},
		OrderVoucher{},
		OrderItem{},
		Payment{},
		Shipment{},
		Review{},
		Discount{},
		ProductDiscount{},
		ProductInformation{},
		Voucher{},
		UserVoucher{},
		ProductAnalytics{},
		Notification{},
		UserNotification{},
	}
}
