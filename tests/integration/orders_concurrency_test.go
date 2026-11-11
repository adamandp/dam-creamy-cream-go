package integration_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/orders"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
)

// Distinct PostgreSQL transactions must compete for the same Inventory row.
// With 3 units available and two buyers of 2 units, only one may commit.
func TestCheckoutConcurrentBuyersCannotOversell(t *testing.T) {
	runConcurrentCheckout(t, 3, false)
}
func TestCheckoutConcurrentBuyersCannotDoubleRedeemVoucher(t *testing.T) {
	runConcurrentCheckout(t, 10, true)
}
func runConcurrentCheckout(t *testing.T, stockQuantity int, withVoucher bool) {
	db := connectPrismaFixture(t)
	role := domain.Role{Name: "Customer"}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	category := domain.Category{Name: "Concurrent Product", IsActive: true}
	if err := db.Create(&category).Error; err != nil {
		t.Fatal(err)
	}
	product := domain.Product{CategoryID: category.ID.ID, Name: "Limited Vanilla", Price: 3000, IsActive: true}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	stock := domain.Inventory{ProductID: product.ID.ID, Quantity: stockQuantity, ReservedQuantity: 0, IsActive: true}
	if err := db.Create(&stock).Error; err != nil {
		t.Fatal(err)
	}
	var voucher domain.Voucher
	if withVoucher {
		value := 1000.0
		voucher = domain.Voucher{Code: "VoucherBounded", Name: "One Remaining", Type: "FIXED", Group: "CHECKOUT",
			Value: &value, IsActive: true, MaxRedeemed: 1, StartDate: time.Now().UTC().Add(-time.Hour),
			EndDate: time.Now().UTC().Add(time.Hour)}
		if err := db.Create(&voucher).Error; err != nil {
			t.Fatal(err)
		}
	}
	users := make([]domain.User, 2)
	tokens := make([]string, 2)
	ses := sessions.NewService(db, "buyer-access", "buyer-refresh")
	now := time.Now().UTC()
	for i := range users {
		users[i] = domain.User{RoleID: role.ID.ID, Username: "CheckoutConcurrent" + string(rune('A'+i)), FullName: "Concurrent Buyer", Password: "test", IsActive: true}
		if err := db.Create(&users[i]).Error; err != nil {
			t.Fatal(err)
		}
		v, err := ses.NewTokens(context.Background(), users[i].ID.ID, role.ID.ID, "", "Fixture", "127.0.0.1")
		if err != nil {
			t.Fatal(err)
		}
		tokens[i] = v.AccessToken
		address := domain.Address{UserID: users[i].ID.ID, RecipientName: "Buyer", RecipientPhoneNumber: "081234567890", Province: "West Java", City: "Bandung", Subdistrict: "Cicendo", Village: "Pajajaran", PostalCode: "40172", Address: "Jalan Merdeka", IsPrimary: true}
		if err := db.Create(&address).Error; err != nil {
			t.Fatal(err)
		}
		cart := domain.Cart{UserID: users[i].ID.ID}
		if err := db.Create(&cart).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&domain.CartItem{CartID: cart.ID.ID, ProductID: product.ID.ID, Quantity: 2}).Error; err != nil {
			t.Fatal(err)
		}
		if withVoucher {
			claim := domain.UserVoucher{UserID: users[i].ID.ID, VoucherID: voucher.ID.ID, Status: "CLAIMED",
				ClaimedAt: now, ExpiredAt: now.Add(time.Hour)}
			if err := db.Create(&claim).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	defer func() {
		var orderIDs []string
		_ = db.Model(&domain.Order{}).Where("user_id IN ?", []string{users[0].ID.ID, users[1].ID.ID}).Pluck("id", &orderIDs).Error
		if len(orderIDs) > 0 {
			_ = db.Where("order_id IN ?", orderIDs).Delete(&domain.Payment{}).Error
			_ = db.Where("order_id IN ?", orderIDs).Delete(&domain.Shipment{}).Error
			_ = db.Where("order_id IN ?", orderIDs).Delete(&domain.OrderItem{}).Error
			_ = db.Where("order_id IN ?", orderIDs).Delete(&domain.OrderVoucher{}).Error
			_ = db.Where("id IN ?", orderIDs).Delete(&domain.Order{}).Error
		}
		var cartIDs []string
		_ = db.Model(&domain.Cart{}).Where("user_id IN ?", []string{users[0].ID.ID, users[1].ID.ID}).Pluck("id", &cartIDs).Error
		if len(cartIDs) > 0 {
			_ = db.Where("cart_id IN ?", cartIDs).Delete(&domain.CartItem{}).Error
		}
		_ = db.Where("id IN ?", cartIDs).Delete(&domain.Cart{}).Error
		if withVoucher {
			_ = db.Where("voucher_id = ?", voucher.ID.ID).Delete(&domain.UserVoucher{}).Error
			_ = db.Delete(&voucher).Error
		}
		_ = db.Delete(&stock).Error
		_ = db.Delete(&product).Error
		_ = db.Delete(&category).Error
		for _, u := range users {
			_ = db.Where("user_id = ?", u.ID.ID).Delete(&domain.UserToken{}).Error
			_ = db.Where("user_id = ?", u.ID.ID).Delete(&domain.Address{}).Error
			_ = db.Delete(&u).Error
		}
		_ = db.Delete(&role).Error
	}()
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	orders.NewService(db, ses).Routes(app)
	codes := make(chan int, 2)
	var wg sync.WaitGroup
	for i := range users {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			var address domain.Address
			if err := db.Where("user_id = ?", users[i].ID.ID).First(&address).Error; err != nil {
				codes <- 0
				return
			}
			in := orders.CheckoutInput{}
			in.Order.UserID = users[i].ID.ID
			if withVoucher {
				in.Order.VoucherIDs = []string{voucher.ID.ID}
			}
			in.Order.AddressID = address.ID.ID
			in.OrderItems = []orders.CheckoutItem{{ProductID: product.ID.ID, Quantity: 2}}
			in.Payment.Method = "BCA"
			in.Shipment.Courier = "jne"
			in.Shipment.Service = "regular"
			in.Shipment.Cost = 0
			in.Shipment.Estimated = now.Add(24 * time.Hour)
			raw, err := json.Marshal(in)
			if err != nil {
				codes <- 0
				return
			}
			req := httptest.NewRequest("POST", "/orders/checkout", strings.NewReader(string(raw)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+tokens[i])
			response, err := app.Test(req)
			if err != nil {
				codes <- 0
				return
			}
			response.Body.Close()
			codes <- response.StatusCode
		}()
	}
	wg.Wait()
	close(codes)
	success, conflict := 0, 0
	for code := range codes {
		if code == 201 {
			success++
		} else if code == 409 {
			conflict++
		} else {
			t.Errorf("unexpected concurrent checkout HTTP %d", code)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("expected one success and one conflict, got 201=%d 409=%d", success, conflict)
	}
	if err := db.First(&stock, "id = ?", stock.ID.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stock.ReservedQuantity != 2 {
		t.Fatalf("oversold inventory: %+v", stock)
	}
	var count int64
	if err := db.Model(&domain.Order{}).Where("user_id IN ?", []string{users[0].ID.ID, users[1].ID.ID}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("expected one committed order, got %d, err %v", count, err)
	}
	if withVoucher {
		var saved domain.Voucher
		if err := db.First(&saved, "id = ?", voucher.ID.ID).Error; err != nil || saved.RedeemedCount != 1 {
			t.Fatalf("voucher double-redeemed: %+v %v", saved, err)
		}
		var used int64
		if err := db.Model(&domain.UserVoucher{}).Where("voucher_id = ? AND status = ?", voucher.ID.ID, "USED").Count(&used).Error; err != nil || used != 1 {
			t.Fatalf("incorrect voucher claimed-status count %d %v", used, err)
		}
	}
}
