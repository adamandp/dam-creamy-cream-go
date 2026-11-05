package integration_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/carts"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/orders"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/products"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
)

// Each case follows the same priced product from public storefront views into
// a real PostgreSQL checkout. A price mismatch must fail this test even when
// Cart, Catalog and Detail agree with one another.
func TestAllDiscountTypesMatchCheckout(t *testing.T) {
	cases := []struct {
		name, typ string
		value     *float64
		wantPrice any // nil for a free-item reward, otherwise final unit price
		paidPrice float64
		gift      bool
	}{
		{"percentage", "PERCENTAGE", float64Pointer(20), float64(8000), 8000, false},
		{"fractional_percentage", "PERCENTAGE", float64Pointer(33.3333), float64(6666.67), 6666.67, false},
		{"fixed", "FIXED", float64Pointer(2000), float64(8000), 8000, false},
		{"fixed_larger_than_price", "FIXED", float64Pointer(15000), float64(0), 0, false},
		{"price_override", "PRICE", float64Pointer(7000), float64(7000), 7000, false},
		{"price_override_zero", "PRICE", float64Pointer(0), float64(0), 0, false},
		{"price_override_above_original", "PRICE", float64Pointer(15000), float64(10000), 10000, false},
		{"free_item_value_present", "FREE_ITEM", float64Pointer(7000), nil, 10000, true},
		{"free_item_value_null", "FREE_ITEM", nil, nil, 10000, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := connectPrismaFixture(t)
			tx := db.Begin()
			if tx.Error != nil {
				t.Fatal(tx.Error)
			}
			defer tx.Rollback()

			role := domain.Role{Name: "Customer"}
			if err := tx.Create(&role).Error; err != nil {
				t.Fatal(err)
			}
			user := domain.User{RoleID: role.ID.ID, Username: "DiscountCaseBuyer", FullName: "Discount Case Buyer", Password: "test", IsActive: true}
			if err := tx.Create(&user).Error; err != nil {
				t.Fatal(err)
			}
			address := domain.Address{UserID: user.ID.ID, RecipientName: "Buyer", RecipientPhoneNumber: "081234567890", Province: "West Java", City: "Bandung", Subdistrict: "Cicendo", Village: "Pajajaran", PostalCode: "40172", Address: "Jalan Merdeka", IsPrimary: true}
			if err := tx.Create(&address).Error; err != nil {
				t.Fatal(err)
			}
			category := domain.Category{Name: "Discount Type Tests", IsActive: true}
			if err := tx.Create(&category).Error; err != nil {
				t.Fatal(err)
			}
			paid := domain.Product{CategoryID: category.ID.ID, Name: "Discount Type Gelato", Price: 10000, IsActive: true}
			if err := tx.Create(&paid).Error; err != nil {
				t.Fatal(err)
			}
			stock := domain.Inventory{ProductID: paid.ID.ID, Quantity: 5, IsActive: true}
			if err := tx.Create(&stock).Error; err != nil {
				t.Fatal(err)
			}
			var reward *domain.Product
			one := 1
			if tc.gift {
				gift := domain.Product{CategoryID: category.ID.ID, Name: "Bonus Gelato", Price: 3000, IsActive: true}
				if err := tx.Create(&gift).Error; err != nil {
					t.Fatal(err)
				}
				if err := tx.Create(&domain.Inventory{ProductID: gift.ID.ID, Quantity: 1, IsActive: true}).Error; err != nil {
					t.Fatal(err)
				}
				reward = &gift
			}
			cart := domain.Cart{UserID: user.ID.ID}
			if err := tx.Create(&cart).Error; err != nil {
				t.Fatal(err)
			}
			if err := tx.Create(&domain.CartItem{CartID: cart.ID.ID, ProductID: paid.ID.ID, Quantity: 1}).Error; err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC()
			discount := domain.Discount{Name: "Price Consistency", DiscountType: tc.typ, Value: tc.value,
				MaxApplied: 1, MaxRedeemed: 10, StartDate: now.Add(-time.Hour), EndDate: now.Add(time.Hour)}
			if reward != nil {
				discount.ProductID = &reward.ID.ID
				discount.Quantity = &one
			}
			if err := tx.Create(&discount).Error; err != nil {
				t.Fatal(err)
			}
			if err := tx.Create(&domain.ProductDiscount{ProductID: paid.ID.ID, DiscountID: discount.ID.ID, IsActive: true}).Error; err != nil {
				t.Fatal(err)
			}

			ses := sessions.NewService(tx, "discount-types-access", "discount-types-refresh")
			token, err := ses.NewTokens(context.Background(), user.ID.ID, role.ID.ID, "", "Test", "127.0.0.1")
			if err != nil {
				t.Fatal(err)
			}
			app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
			carts.NewService(tx, ses).Routes(app)
			products.NewService(tx, stubImageStore{}, ses).Routes(app)
			orders.NewService(tx, ses).Routes(app)
			readDiscount := func(path, cookie string) any {
				t.Helper()
				body := testBody(t, testCall(t, app, "GET", path, "", cookie, 200))
				if values, ok := body["data"].([]any); ok {
					if len(values) != 1 {
						t.Fatalf("expected one product at %s; got %+v", path, values)
					}
					return values[0].(map[string]any)["discountPrice"]
				}
				return body["data"].(map[string]any)["discountPrice"]
			}
			views := map[string]any{
				"cart":    readDiscount("/carts/user", "refreshToken="+token.RefreshToken),
				"catalog": readDiscount("/products/catalog?q=Discount%20Type%20Gelato", ""),
				"detail":  readDiscount("/product/"+paid.ID.ID, ""),
			}
			for view, got := range views {
				if got != tc.wantPrice {
					t.Fatalf("%s %s: discountPrice=%v, expected %v", tc.name, view, got, tc.wantPrice)
				}
			}

			input := orders.CheckoutInput{}
			input.Order.UserID = user.ID.ID
			input.Order.AddressID = address.ID.ID
			input.OrderItems = []orders.CheckoutItem{{ProductID: paid.ID.ID, DiscountID: &discount.ID.ID, Quantity: 1}}
			input.Payment.Method = "BCA"
			input.Shipment.Courier = "jne"
			input.Shipment.Service = "regular"
			input.Shipment.Cost = 0
			input.Shipment.Estimated = now.Add(24 * time.Hour)
			body, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			response := testBody(t, testCallAs(t, app, token.AccessToken, "POST", "/orders/checkout", string(body), "", 201))
			data := response["data"].(map[string]any)
			if data["finalPrice"] != tc.paidPrice {
				t.Fatalf("%s checkout finalPrice=%v, expected %v; storefront=%+v", tc.name, data["finalPrice"], tc.paidPrice, views)
			}
			var rows []domain.OrderItem
			if err := tx.Where("order_id = ?", data["orderId"].(string)).Find(&rows).Error; err != nil {
				t.Fatal(err)
			}
			wantRows := 1
			if tc.gift {
				wantRows = 2
			}
			if len(rows) != wantRows {
				t.Fatalf("%s expected %d lines, got %+v", tc.name, wantRows, rows)
			}
			for _, row := range rows {
				switch row.ProductID {
				case paid.ID.ID:
					if row.FinalPrice != tc.paidPrice {
						t.Fatalf("paid product price mismatch: %+v", row)
					}
				default:
					if !tc.gift || reward == nil || row.ProductID != reward.ID.ID || row.Price != 0 || row.FinalPrice != 0 {
						t.Fatalf("unexpected reward line: %+v", row)
					}
				}
			}
			if tc.gift {
				var rewardStock domain.Inventory
				if err := tx.Where("product_id = ?", reward.ID.ID).First(&rewardStock).Error; err != nil {
					t.Fatal(err)
				}
				if rewardStock.ReservedQuantity != 1 {
					t.Fatalf("reward stock not reserved: %+v", rewardStock)
				}
			}
		})
	}
}

func float64Pointer(v float64) *float64 { return &v }
