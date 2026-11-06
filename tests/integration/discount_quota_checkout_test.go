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

// The same valid discount must disappear from Cart, Catalog and Product Detail
// immediately when its quota is exhausted, without waiting for expiration.
// Checkout remains authoritative: it locks the discount and rejects a
// competing/exhausted redemption even if a storefront quote was just shown.
func TestQuotaExhaustionAcrossStorefrontAndCheckout(t *testing.T) {
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
	user := domain.User{RoleID: role.ID.ID, Username: "QuotaBuyer", FullName: "Quota Buyer", Password: "test", IsActive: true}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	address := domain.Address{UserID: user.ID.ID, RecipientName: "Quota", RecipientPhoneNumber: "081234567890", Province: "West Java", City: "Bandung", Subdistrict: "Cicendo", Village: "Pajajaran", PostalCode: "40172", Address: "Jalan Merdeka", IsPrimary: true}
	if err := tx.Create(&address).Error; err != nil {
		t.Fatal(err)
	}
	category := domain.Category{Name: "Quota Desserts", IsActive: true}
	if err := tx.Create(&category).Error; err != nil {
		t.Fatal(err)
	}
	product := domain.Product{CategoryID: category.ID.ID, Name: "Quota Gelato", Price: 10000, IsActive: true}
	if err := tx.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	stock := domain.Inventory{ProductID: product.ID.ID, Quantity: 4, IsActive: true}
	if err := tx.Create(&stock).Error; err != nil {
		t.Fatal(err)
	}
	cart := domain.Cart{UserID: user.ID.ID}
	if err := tx.Create(&cart).Error; err != nil {
		t.Fatal(err)
	}
	line := domain.CartItem{CartID: cart.ID.ID, ProductID: product.ID.ID, Quantity: 1}
	if err := tx.Create(&line).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	override := 7000.0
	discount := domain.Discount{Name: "Quota Price", DiscountType: "PRICE", Value: &override, MaxApplied: 1, MaxRedeemed: 2, RedeemedCount: 1, StartDate: now.Add(-time.Hour), EndDate: now.Add(time.Hour)}
	if err := tx.Create(&discount).Error; err != nil {
		t.Fatal(err)
	}
	assignment := domain.ProductDiscount{ProductID: product.ID.ID, DiscountID: discount.ID.ID, IsActive: true}
	if err := tx.Create(&assignment).Error; err != nil {
		t.Fatal(err)
	}
	ses := sessions.NewService(tx, "quota-access", "quota-refresh")
	token, err := ses.NewTokens(context.Background(), user.ID.ID, role.ID.ID, "", "Fixture", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	carts.NewService(tx, ses).Routes(app)
	products.NewService(tx, stubImageStore{}, ses).Routes(app)
	orders.NewService(tx, ses).Routes(app)

	read := func(path, cookie string) any {
		t.Helper()
		body := testBody(t, testCall(t, app, "GET", path, "", cookie, 200))
		if rows, ok := body["data"].([]any); ok {
			if len(rows) != 1 {
				t.Fatalf("%s: wanted one row, got %v", path, rows)
			}
			return rows[0].(map[string]any)["discountPrice"]
		}
		return body["data"].(map[string]any)["discountPrice"]
	}
	check := func(label string, want any) {
		t.Helper()
		got := map[string]any{
			"cart":    read("/carts/user", "refreshToken="+token.RefreshToken),
			"catalog": read("/products/catalog?q=Quota%20Gelato", ""),
			"detail":  read("/product/"+product.ID.ID, ""),
		}
		for view, price := range got {
			if price != want {
				t.Fatalf("%s %s discountPrice=%v want=%v", label, view, price, want)
			}
		}
	}
	input := orders.CheckoutInput{}
	input.Order.UserID = user.ID.ID
	input.Order.AddressID = address.ID.ID
	input.OrderItems = []orders.CheckoutItem{{ProductID: product.ID.ID, DiscountID: &discount.ID.ID, Quantity: 1}}
	input.Payment.Method = "BCA"
	input.Shipment.Courier = "jne"
	input.Shipment.Service = "regular"
	input.Shipment.Estimated = now.Add(24 * time.Hour)
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	checkout := func(code int) map[string]any {
		return testBody(t, testCallAs(t, app, token.AccessToken, "POST", "/orders/checkout", string(raw), "", code))
	}

	check("one redemption available", float64(7000))
	if err := tx.Model(&discount).Update("redeemed_count", 2).Error; err != nil {
		t.Fatal(err)
	}
	check("quota exhausted", nil)
	rejected := checkout(409)
	if rejected["message"] != "Discount redemption limit exceeded" {
		t.Fatalf("wrong rejection: %+v", rejected)
	}
	var ordersBefore int64
	if err := tx.Model(&domain.Order{}).Where("user_id = ?", user.ID.ID).Count(&ordersBefore).Error; err != nil || ordersBefore != 0 {
		t.Fatalf("exhausted promo created order: %v count=%d", err, ordersBefore)
	}
	if err := tx.Model(&discount).Update("redeemed_count", 1).Error; err != nil {
		t.Fatal(err)
	}
	check("quota reopened", float64(7000))
	created := checkout(201)
	if created["data"].(map[string]any)["finalPrice"] != float64(7000) {
		t.Fatalf("checkout price mismatch: %+v", created)
	}
	if err := tx.First(&discount, "id = ?", discount.ID.ID).Error; err != nil || discount.RedeemedCount != 2 {
		t.Fatalf("checkout did not consume final redemption: %v %+v", err, discount)
	}
	// A successful checkout removes the bought CartItem. Re-add it to verify
	// the subsequent Cart view also hides the now-exhausted promotion.
	if err := tx.Create(&domain.CartItem{CartID: cart.ID.ID, ProductID: product.ID.ID, Quantity: 1}).Error; err != nil {
		t.Fatal(err)
	}
	check("quota consumed by checkout", nil)
	if err := tx.First(&stock, "id = ?", stock.ID.ID).Error; err != nil || stock.ReservedQuantity != 1 {
		t.Fatalf("checkout stock reservation wrong: %v %+v", err, stock)
	}
}
