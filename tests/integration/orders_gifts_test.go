package integration_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/orders"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
)

func TestCheckoutFreeGiftRewardsAlsoReserveStock(t *testing.T) {
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
	user := domain.User{RoleID: role.ID.ID, Username: "FreeRewardBuyer", FullName: "Reward Buyer", Password: "test", IsActive: true}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	address := domain.Address{UserID: user.ID.ID, RecipientName: "Buyer", RecipientPhoneNumber: "081234567890", Province: "West Java", City: "Bandung", Subdistrict: "Cicendo", Village: "Pajajaran", PostalCode: "40172", Address: "Jalan Merdeka", IsPrimary: true}
	if err := tx.Create(&address).Error; err != nil {
		t.Fatal(err)
	}
	cat := domain.Category{Name: "Gift Category", IsActive: true}
	if err := tx.Create(&cat).Error; err != nil {
		t.Fatal(err)
	}
	paid := domain.Product{CategoryID: cat.ID.ID, Name: "Paid Ice Cream", Price: 5000, IsActive: true}
	gift := domain.Product{CategoryID: cat.ID.ID, Name: "Bonus Ice Cream", Price: 2000, IsActive: true}
	for _, p := range []*domain.Product{&paid, &gift} {
		if err := tx.Create(p).Error; err != nil {
			t.Fatal(err)
		}
	}
	paidStock := domain.Inventory{ProductID: paid.ID.ID, Quantity: 2, IsActive: true}
	giftStock := domain.Inventory{ProductID: gift.ID.ID, Quantity: 1, IsActive: true}
	for _, inv := range []*domain.Inventory{&paidStock, &giftStock} {
		if err := tx.Create(inv).Error; err != nil {
			t.Fatal(err)
		}
	}
	cart := domain.Cart{UserID: user.ID.ID}
	if err := tx.Create(&cart).Error; err != nil {
		t.Fatal(err)
	}
	ci := domain.CartItem{CartID: cart.ID.ID, ProductID: paid.ID.ID, Quantity: 1}
	if err := tx.Create(&ci).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	qty := 1
	discount := domain.Discount{Name: "Free Product Discount", DiscountType: "FREE_ITEM", ProductID: &gift.ID.ID, Quantity: &qty, MaxApplied: 1, MaxRedeemed: 10, StartDate: now.Add(-time.Hour), EndDate: now.Add(time.Hour)}
	if err := tx.Create(&discount).Error; err != nil {
		t.Fatal(err)
	}
	association := domain.ProductDiscount{ProductID: paid.ID.ID, DiscountID: discount.ID.ID, IsActive: true}
	if err := tx.Create(&association).Error; err != nil {
		t.Fatal(err)
	}
	voucher := domain.Voucher{Code: "BONUS501", Name: "Bonus Voucher", Type: "FREE_ITEM", Group: "CHECKOUT", ProductID: &gift.ID.ID, Quantity: &qty, IsActive: true, StartDate: now.Add(-time.Hour), EndDate: now.Add(time.Hour), MaxRedeemed: 1}
	if err := tx.Create(&voucher).Error; err != nil {
		t.Fatal(err)
	}
	claim := domain.UserVoucher{UserID: user.ID.ID, VoucherID: voucher.ID.ID, Status: "CLAIMED", ClaimedAt: now, ExpiredAt: now.Add(time.Hour)}
	if err := tx.Create(&claim).Error; err != nil {
		t.Fatal(err)
	}
	ses := sessions.NewService(tx, "bonus-access", "bonus-refresh")
	token, err := ses.NewTokens(context.Background(), user.ID.ID, role.ID.ID, "", "Fixture", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	orders.NewService(tx, ses).Routes(app)
	input := orders.CheckoutInput{}
	input.Order.UserID = user.ID.ID
	input.Order.AddressID = address.ID.ID
	input.Order.VoucherIDs = []string{voucher.ID.ID}
	input.OrderItems = []orders.CheckoutItem{{ProductID: paid.ID.ID, DiscountID: &discount.ID.ID, Quantity: 1}}
	input.Shipment.Courier = "jne"
	input.Shipment.Service = "regular"
	input.Shipment.Estimated = now.Add(48 * time.Hour)
	input.Payment.Method = "BCA"
	checkout := func(want int) map[string]any {
		bytes, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		return testBody(t, testCallAs(t, app, token.AccessToken, "POST", "/orders/checkout", string(bytes), "", want))
	}
	checkout(409) // only 1 gift in stock, but 2 gift sources
	var count int64
	if err := tx.Model(&domain.Order{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("failed gift checkout left order: %v %d", err, count)
	}
	if err := tx.Model(&giftStock).Update("quantity", 2).Error; err != nil {
		t.Fatal(err)
	}
	out := checkout(201)
	orderID := out["data"].(map[string]any)["orderId"].(string)
	var rows []domain.OrderItem
	if err := tx.Where("order_id = ?", orderID).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 1 paid + 2 free reward lines: %+v", rows)
	}
	for _, row := range rows {
		if row.ProductID == gift.ID.ID && (row.Price != 0 || row.FinalPrice != 0 || row.Quantity != 1) {
			t.Fatalf("gift was not free: %+v", row)
		}
	}
	if err := tx.First(&giftStock, "id = ?", giftStock.ID.ID).Error; err != nil || giftStock.ReservedQuantity != 2 {
		t.Fatalf("gift inventory reservation wrong: %v %+v", err, giftStock)
	}
	if err := tx.First(&paidStock, "id = ?", paidStock.ID.ID).Error; err != nil || paidStock.ReservedQuantity != 1 {
		t.Fatalf("paid inventory wrong: %v %+v", err, paidStock)
	}
	if err := tx.First(&claim, "id = ?", claim.ID.ID).Error; err != nil || claim.Status != "USED" {
		t.Fatalf("voucher not redeemed: %v %+v", err, claim)
	}
	if out["data"].(map[string]any)["finalPrice"] != float64(5000) {
		t.Fatalf("unexpected free-gift total: %+v", out)
	}
}
