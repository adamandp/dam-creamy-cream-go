package model_test

import (
	"sync"
	"testing"

	"dam-creamy-cream-go/internal/dal/model"

	"gorm.io/gorm/schema"
)

func TestModelTableAndColumnMapping(t *testing.T) {
	expected := map[string]string{
		"Role": "roles", "User": "users", "UserToken": "user_tokens",
		"Address": "addresses", "Product": "products", "Category": "category",
		"Inventory": "inventory", "Cart": "carts", "CartItem": "cart_items",
		"Order": "orders", "OrderVoucher": "order_vouchers", "OrderItem": "order_items",
		"Payment": "payments", "Shipment": "shipments", "Review": "reviews",
		"Discount": "discounts", "ProductDiscount": "product_discounts",
		"ProductInformation": "product_information", "Voucher": "vouchers",
		"UserVoucher": "user_vouchers", "ProductAnalytics": "product_analytics",
		"Notification": "notifications", "UserNotification": "user_notifications",
	}
	models := model.All()
	if len(models) != 23 {
		t.Fatalf("models = %d, want 23", len(models))
	}
	seen := make(map[string]bool)
	for _, m := range models {
		s, err := schema.Parse(m, &sync.Map{}, schema.NamingStrategy{})
		if err != nil {
			t.Fatalf("parse %T: %v", m, err)
		}
		want, ok := expected[s.Name]
		if !ok || s.Table != want {
			t.Errorf("%s table = %q, want %q", s.Name, s.Table, want)
		}
		if seen[s.Name] {
			t.Errorf("duplicate model %s", s.Name)
		}
		seen[s.Name] = true
	}
	for _, tc := range []struct {
		m      any
		column string
		kind   string
	}{
		{model.UserToken{}, "browserInfo", "text"},
		{model.UserToken{}, "expiredAt", "timestamp(3)"},
		{model.OrderItem{}, "finalPrice", "double precision"},
		{model.ProductInformation{}, "highlights", "jsonb"},
		{model.ProductInformation{}, "serving_suggestions", "jsonb"},
		{model.ProductInformation{}, "product_details", "jsonb"},
		{model.Category{}, "is_active", "boolean"},
	} {
		s, err := schema.Parse(tc.m, &sync.Map{}, schema.NamingStrategy{})
		if err != nil {
			t.Fatal(err)
		}
		f := s.LookUpField(tc.column)
		if f == nil || f.TagSettings["TYPE"] != tc.kind {
			t.Errorf("%s.%s type tag mismatch: %v", s.Name, tc.column, f)
		}
	}
	s, err := schema.Parse(model.OrderVoucher{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatal(err)
	}
	if len(s.PrimaryFields) != 2 {
		t.Errorf("OrderVoucher primary keys = %d, want 2", len(s.PrimaryFields))
	}
}
