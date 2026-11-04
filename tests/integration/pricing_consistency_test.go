package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/carts"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/products"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
)

// Original regressions: inactive discounts applied in Cart; expired discounts
// applied in Catalog/Detail. All three reads must agree at any given time.
func TestCartCatalogDetailDiscountEligibility(t *testing.T) {
	db := connectPrismaFixture(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	role := domain.Role{Name: "PriceCustomer"}
	if err := tx.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := domain.User{RoleID: role.ID.ID, Username: "PricingFixture", FullName: "Pricing Fixture", Password: "test", IsActive: true}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	cat := domain.Category{Name: "Pricing Desserts", IsActive: true}
	if err := tx.Create(&cat).Error; err != nil {
		t.Fatal(err)
	}
	p := domain.Product{CategoryID: cat.ID.ID, Name: "Pricing Gelato", Price: 12000, IsActive: true}
	if err := tx.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	percent := 50.0
	now := time.Now().UTC()
	d := domain.Discount{Name: "Pricing Promo", DiscountType: "PERCENTAGE", Value: &percent, MaxApplied: 1, MaxRedeemed: 10, StartDate: now.Add(-2 * time.Hour), EndDate: now.Add(2 * time.Hour)}
	if err := tx.Create(&d).Error; err != nil {
		t.Fatal(err)
	}
	association := domain.ProductDiscount{ProductID: p.ID.ID, DiscountID: d.ID.ID, IsActive: false}
	if err := tx.Create(&association).Error; err != nil {
		t.Fatal(err)
	}
	cart := domain.Cart{UserID: user.ID.ID}
	if err := tx.Create(&cart).Error; err != nil {
		t.Fatal(err)
	}
	line := domain.CartItem{CartID: cart.ID.ID, ProductID: p.ID.ID, Quantity: 1}
	if err := tx.Create(&line).Error; err != nil {
		t.Fatal(err)
	}
	ses := sessions.NewService(tx, "pricing-access", "pricing-refresh")
	tokens, err := ses.NewTokens(context.Background(), user.ID.ID, role.ID.ID, "", "Test", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	carts.NewService(tx, ses).Routes(app)
	products.NewService(tx, stubImageStore{}, ses).Routes(app)
	discountPrice := func(path, cookie string) any {
		body := testBody(t, testCall(t, app, "GET", path, "", cookie, 200))
		var row map[string]any
		if v, ok := body["data"].([]any); ok {
			row = v[0].(map[string]any)
		} else {
			row = body["data"].(map[string]any)
		}
		return row["discountPrice"]
	}
	cartPrice := func() any { return discountPrice("/carts/user", "refreshToken="+tokens.RefreshToken) }
	catalogPrice := func() any { return discountPrice("/products/catalog?q=Pricing", "") }
	detailPrice := func() any { return discountPrice("/product/"+p.ID.ID, "") }
	check := func(label string, want any) {
		t.Helper()
		for name, actual := range map[string]any{"cart": cartPrice(), "catalog": catalogPrice(), "detail": detailPrice()} {
			if actual != want {
				t.Fatalf("%s: %s discountPrice=%v, expected=%v", label, name, actual, want)
			}
		}
	}
	check("inactive association", nil)
	if err := tx.Model(&association).Update("is_active", true).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Model(&d).Update("end_date", now.Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	check("expired discount", nil)
	if err := tx.Model(&d).Updates(map[string]any{"start_date": now.Add(time.Hour), "end_date": now.Add(2 * time.Hour)}).Error; err != nil {
		t.Fatal(err)
	}
	check("future discount", nil)
	if err := tx.Model(&d).Update("start_date", now.Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	check("currently active percentage", float64(6000))
	fixed := 15000.0
	if err := tx.Model(&d).Updates(map[string]any{"discount_type": "FIXED", "value": fixed}).Error; err != nil {
		t.Fatal(err)
	}
	check("fixed discount floor zero", float64(0))
	if err := tx.Model(&d).Update("value", float64(0)).Error; err != nil {
		t.Fatal(err)
	}
	check("zero-value discount", nil)
}
