package integration_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/products"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
)

func TestCatalogAndProductInformationOnPrismaSchema(t *testing.T) {
	db := connectPrismaFixture(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()

	cat := domain.Category{Name: "Ice Cream", IsActive: true}
	if err := tx.Create(&cat).Error; err != nil {
		t.Fatal(err)
	}
	description := "Great ice cream"
	product := domain.Product{CategoryID: cat.ID.ID, Name: "Vanilla", Description: &description, Price: 12000, IsActive: true}
	if err := tx.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	discountValue := 20.0
	discount := domain.Discount{Name: "Special", DiscountType: "PERCENTAGE", Value: &discountValue, MaxApplied: 1, MaxRedeemed: 20, StartDate: time.Now().UTC().Add(-time.Hour), EndDate: time.Now().UTC().Add(time.Hour)}
	if err := tx.Create(&discount).Error; err != nil {
		t.Fatal(err)
	}
	pd := domain.ProductDiscount{ProductID: product.ID.ID, DiscountID: discount.ID.ID, IsActive: true}
	if err := tx.Create(&pd).Error; err != nil {
		t.Fatal(err)
	}
	info := domain.ProductInformation{
		ProductID:          product.ID.ID,
		Highlights:         json.RawMessage(`["Rich and creamy"]`),
		ServingSuggestions: json.RawMessage(`["Serve cold"]`),
		ProductDetails:     json.RawMessage(`{"size":"500ml","storage":"Freezer","shelfLife":"1 year"}`),
		WhyChoose:          "Great flavor",
	}
	if err := tx.Create(&info).Error; err != nil {
		t.Fatal(err)
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	products.NewService(tx, stubImageStore{}, sessions.NewService(tx, "test-access", "test-refresh")).Routes(app)

	catalog := testBody(t, testCall(t, app, "GET", "/products/catalog?q=vanilla&cat="+cat.ID.ID+"&min=10000&max=15000&sort=asc", "", "", 200))
	rows, ok := catalog["data"].([]any)
	if !ok || len(rows) != 1 {
		t.Fatalf("catalog rows unexpected: %v", catalog)
	}
	p, ok := rows[0].(map[string]any)
	if !ok || p["discountPrice"] != 9600.0 || p["discountType"] != "PERCENTAGE" || p["discountValue"] != 20.0 {
		t.Errorf("catalog discount mapping mismatch: %v", p)
	}
	// ProductQuerySchema is strict and Zod distinguishes missing from empty.
	for _, tc := range []struct {
		query  string
		status int
	}{
		{"q=%20", 200}, // whitespace is a valid one-character Zod string
		{"min=", 200}, // z.coerce.number("") == 0
		{"max=", 404}, // a supplied zero max excludes this positive-price product
		{"q=", 400},
		{"sort=", 400},
		{"cat=", 400},
		{"unknown=ignored-in-first-pass", 400},
	} {
		testBody(t, testCall(t, app, "GET", "/products/catalog?"+tc.query, "", "", tc.status))
	}
	productID := product.ID.ID
	detail := testBody(t, testCall(t, app, "GET", "/product/"+productID, "", "", 200))
	d, ok := detail["data"].(map[string]any)
	if !ok || d["category"] != "Ice Cream" || d["rate"] != 0.0 || d["sales"] != 0.0 {
		t.Fatalf("product detail mismatch: %v", detail)
	}
	parsed := testBody(t, testCall(t, app, "GET", "/product/information/"+productID, "", "", 200))
	data, ok := parsed["data"].(map[string]any)
	if !ok {
		t.Fatalf("invalid product information payload %v", parsed)
	}
	expected := `Great ice cream

🍦 Highlights
• Rich and creamy

🍨 Serving Suggestions
• Serve cold

📦 Product Details
• Size: 500ml
• Storage: Freezer
• Shelf life: 1 year

⚡ Why Choose This Product?
Great flavor`
	if data["information"] != expected {
		t.Errorf("product information differs from expected string: %q", data["information"])
	}
	// Query layer should not return inactive products.
	if err := tx.Model(&product).Update("is_active", false).Error; err != nil {
		t.Fatal(err)
	}
	res, err := app.Test(httptest.NewRequest("GET", "/products/catalog?q=vanilla", nil))
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != 404 {
		t.Errorf("inactive products should not be visible in first catalog page, got %d", res.StatusCode)
	}
}
