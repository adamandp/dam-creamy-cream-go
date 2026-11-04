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

func TestCheckoutAtomicPricingVoucherInventoryAndOwner(t *testing.T) {
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
	alice := domain.User{RoleID: role.ID.ID, Username: "CheckoutAlice", FullName: "Checkout Alice", Password: "fixture", IsActive: true}
	bob := domain.User{RoleID: role.ID.ID, Username: "CheckoutBob", FullName: "Checkout Bob", Password: "fixture", IsActive: true}
	for _, u := range []*domain.User{&alice, &bob} {
		if err := tx.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	address := domain.Address{UserID: alice.ID.ID, RecipientName: "Alice", RecipientPhoneNumber: "081234567890", Province: "West Java", City: "Bandung", Subdistrict: "Cicendo", Village: "Pajajaran", PostalCode: "40172", Address: "Jalan Merdeka", IsPrimary: true}
	foreign := address
	foreign.ID.ID = ""
	foreign.UserID = bob.ID.ID
	for _, a := range []*domain.Address{&address, &foreign} {
		if err := tx.Create(a).Error; err != nil {
			t.Fatal(err)
		}
	}
	category := domain.Category{Name: "Checkout Desserts", IsActive: true}
	if err := tx.Create(&category).Error; err != nil {
		t.Fatal(err)
	}
	product := domain.Product{CategoryID: category.ID.ID, Name: "Ice Cream", Price: 12000, IsActive: true}
	if err := tx.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	inventory := domain.Inventory{ProductID: product.ID.ID, Quantity: 10, ReservedQuantity: 1, IsActive: true}
	if err := tx.Create(&inventory).Error; err != nil {
		t.Fatal(err)
	}
	cart := domain.Cart{UserID: alice.ID.ID}
	if err := tx.Create(&cart).Error; err != nil {
		t.Fatal(err)
	}
	ci := domain.CartItem{CartID: cart.ID.ID, ProductID: product.ID.ID, Quantity: 4}
	if err := tx.Create(&ci).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	value := 2000.0
	discount := domain.Discount{Name: "Simple Fixed Discount", DiscountType: "FIXED", Value: &value, MaxApplied: 10, MaxRedeemed: 20, StartDate: now.Add(-time.Hour), EndDate: now.Add(time.Hour)}
	if err := tx.Create(&discount).Error; err != nil {
		t.Fatal(err)
	}
	association := domain.ProductDiscount{ProductID: product.ID.ID, DiscountID: discount.ID.ID, IsActive: true}
	if err := tx.Create(&association).Error; err != nil {
		t.Fatal(err)
	}
	fiveThousand, fifty := 5000.0, 50.0
	checkoutVoucher := domain.Voucher{Code: "CHK501", Name: "Checkout Voucher", Type: "FIXED", Group: "CHECKOUT", Value: &fiveThousand, IsActive: true, StartDate: now.Add(-time.Hour), EndDate: now.Add(time.Hour), MaxRedeemed: 10}
	shippingVoucher := domain.Voucher{Code: "SHP501", Name: "Shipping Voucher", Type: "PERCENTAGE", Group: "SHIPPING", Value: &fifty, IsActive: true, StartDate: now.Add(-time.Hour), EndDate: now.Add(time.Hour), MaxRedeemed: 10}
	for _, v := range []*domain.Voucher{&checkoutVoucher, &shippingVoucher} {
		if err := tx.Create(v).Error; err != nil {
			t.Fatal(err)
		}
	}
	claims := []domain.UserVoucher{
		{UserID: alice.ID.ID, VoucherID: checkoutVoucher.ID.ID, Status: "CLAIMED", ClaimedAt: now, ExpiredAt: now.Add(time.Hour)},
		{UserID: alice.ID.ID, VoucherID: shippingVoucher.ID.ID, Status: "CLAIMED", ClaimedAt: now, ExpiredAt: now.Add(time.Hour)},
	}
	for i := range claims {
		if err := tx.Create(&claims[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	ses := sessions.NewService(tx, "checkout-access", "checkout-refresh")
	tokens, err := ses.NewTokens(context.Background(), alice.ID.ID, role.ID.ID, "", "Fixture", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	bobTokens, err := ses.NewTokens(context.Background(), bob.ID.ID, role.ID.ID, "", "Fixture", "127.0.0.2")
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	orders.NewService(tx, ses).Routes(app)
	input := orders.CheckoutInput{}
	input.Order.UserID = alice.ID.ID
	input.Order.AddressID = address.ID.ID
	input.Order.VoucherIDs = []string{checkoutVoucher.ID.ID, shippingVoucher.ID.ID}
	input.OrderItems = []orders.CheckoutItem{{ProductID: product.ID.ID, DiscountID: &discount.ID.ID, Quantity: 2}}
	input.Payment.Method = "BCA"
	input.Shipment.Courier = "jne"
	input.Shipment.Service = "regular"
	input.Shipment.Cost = 8000
	input.Shipment.Estimated = now.Add(48 * time.Hour)
	toJSON := func(v orders.CheckoutInput) string {
		b, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		return string(b)
	}
	checkout := func(v orders.CheckoutInput, token string, want int) map[string]any {
		return testBody(t, testCallAs(t, app, token, "POST", "/orders/checkout", toJSON(v), "", want))
	}
	// Public requests, cross-user address and unclaimed vouchers cannot create orders.
	checkout(input, "", 401)
	checkout(input, bobTokens.AccessToken, 403)
	bad := input
	bad.Order.AddressID = foreign.ID.ID
	checkout(bad, tokens.AccessToken, 409)
	bad = input
	bad.Order.UserID = bob.ID.ID
	checkout(bad, tokens.AccessToken, 403)
	bad = input
	bad.Order.VoucherIDs = []string{checkoutVoucher.ID.ID, checkoutVoucher.ID.ID}
	checkout(bad, tokens.AccessToken, 400)
	bad = input
	bad.OrderItems = []orders.CheckoutItem{{ProductID: product.ID.ID, Quantity: 9}}
	checkout(bad, tokens.AccessToken, 409) // cart only has 4
	bad = input
	bad.Order.VoucherIDs = []string{}
	bad.OrderItems = []orders.CheckoutItem{{ProductID: product.ID.ID, Quantity: 2}}
	if err := tx.Model(&inventory).Update("reserved_quantity", 9).Error; err != nil {
		t.Fatal(err)
	}
	checkout(bad, tokens.AccessToken, 409)
	if err := tx.Model(&inventory).Update("reserved_quantity", 1).Error; err != nil {
		t.Fatal(err)
	}
	var before int64
	if err := tx.Model(&domain.Order{}).Where("user_id = ?", alice.ID.ID).Count(&before).Error; err != nil || before != 0 {
		t.Fatalf("failed attempts persisted orders: %v %d", err, before)
	}
	output := checkout(input, tokens.AccessToken, 201)
	result := output["data"].(map[string]any)
	if result["totalPrice"] != float64(24000) || result["discountAmount"] != float64(4000) || result["voucherAmount"] != float64(9000) || result["shippingCost"] != float64(4000) || result["finalPrice"] != float64(19000) {
		t.Fatalf("checkout totals wrong: %v", result)
	}
	orderID := result["orderId"].(string)
	var stored domain.Order
	if err := tx.First(&stored, "id = ?", orderID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != "PENDING" || stored.FinalPrice != 19000 || stored.VoucherAmount != 9000 {
		t.Fatalf("order was not persisted correctly %+v", stored)
	}
	var payment domain.Payment
	var shipment domain.Shipment
	if err := tx.Where("order_id = ?", orderID).First(&payment).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Where("order_id = ?", orderID).First(&shipment).Error; err != nil {
		t.Fatal(err)
	}
	if payment.Amount != 19000 || payment.Status != "PENDING" || payment.Method != "BCA" || shipment.Cost != 4000 || shipment.Status != "PENDING" {
		t.Fatalf("pending records wrong payment=%+v shipment=%+v", payment, shipment)
	}
	var items []domain.OrderItem
	if err := tx.Where("order_id = ?", orderID).Find(&items).Error; err != nil || len(items) != 1 || items[0].FinalPrice != 10000 || items[0].Quantity != 2 {
		t.Fatalf("order items wrong: %v %+v", err, items)
	}
	if err := tx.First(&inventory, "id = ?", inventory.ID.ID).Error; err != nil || inventory.ReservedQuantity != 3 {
		t.Fatalf("inventory reservation mismatch: %v %+v", err, inventory)
	}
	if err := tx.First(&ci, "id = ?", ci.ID.ID).Error; err != nil || ci.Quantity != 2 {
		t.Fatalf("cart decrement mismatch: %v %+v", err, ci)
	}
	if err := tx.First(&discount, "id = ?", discount.ID.ID).Error; err != nil || discount.RedeemedCount != 2 {
		t.Fatalf("discount tally wrong %v %+v", err, discount)
	}
	for _, v := range []domain.Voucher{checkoutVoucher, shippingVoucher} {
		var saved domain.Voucher
		if err := tx.First(&saved, "id = ?", v.ID.ID).Error; err != nil || saved.RedeemedCount != 1 {
			t.Fatalf("voucher tally wrong %v %+v", err, saved)
		}
		var claim domain.UserVoucher
		if err := tx.Where("voucher_id = ? AND user_id = ?", v.ID.ID, alice.ID.ID).First(&claim).Error; err != nil || claim.Status != "USED" {
			t.Fatalf("voucher claim not consumed %v %+v", err, claim)
		}
	}
	// Inject a database failure at the last checkout step (Shipment INSERT).
	// GORM must rollback Order, OrderItems, Payment, cart changes and inventory
	// reservations from the nested checkout transaction, not leave partial rows.
	if err := tx.Exec(`CREATE FUNCTION pr3_reject_shipment() RETURNS trigger AS $$
	BEGIN RAISE EXCEPTION 'expected integration shipment failure'; END;
	$$ LANGUAGE plpgsql`).Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec(`CREATE TRIGGER pr3_checkout_rollback BEFORE INSERT ON shipments
	FOR EACH ROW EXECUTE FUNCTION pr3_reject_shipment()`).Error; err != nil {
		t.Fatal(err)
	}
	late := input
	late.Order.VoucherIDs = nil
	late.OrderItems = []orders.CheckoutItem{{ProductID: product.ID.ID, Quantity: 1}}
	checkout(late, tokens.AccessToken, 500)
	var postFailureCount int64
	if err := tx.Model(&domain.Order{}).Where("user_id = ?", alice.ID.ID).Count(&postFailureCount).Error; err != nil || postFailureCount != 1 {
		t.Fatalf("shipment failure left partial order: %v count=%d", err, postFailureCount)
	}
	var postInventory domain.Inventory
	var postCart domain.CartItem
	if err := tx.First(&postInventory, "id = ?", inventory.ID.ID).Error; err != nil || postInventory.ReservedQuantity != 3 {
		t.Fatalf("shipment failure leaked inventory reservation: %v %+v", err, postInventory)
	}
	if err := tx.First(&postCart, "id = ?", ci.ID.ID).Error; err != nil || postCart.Quantity != 2 {
		t.Fatalf("shipment failure consumed cart: %v %+v", err, postCart)
	}
	if err := tx.Exec("DROP TRIGGER pr3_checkout_rollback ON shipments").Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Exec("DROP FUNCTION pr3_reject_shipment()").Error; err != nil {
		t.Fatal(err)
	}
	checkout(input, tokens.AccessToken, 409) // voucher already used; reject replay
	var after int64
	if err := tx.Model(&domain.Order{}).Where("user_id = ?", alice.ID.ID).Count(&after).Error; err != nil || after != 1 {
		t.Fatalf("replay persisted order: %v %d", err, after)
	}
	// Details and user order listing remain owner-only.
	details := testBody(t, testCallAs(t, app, tokens.AccessToken, "GET", "/orders/"+orderID, "", "", 200))
	if details["data"] == nil {
		t.Fatal("missing order detail")
	}
	testCallAs(t, app, bobTokens.AccessToken, "GET", "/orders/"+orderID, "", "", 403).Body.Close()
	bobList := testBody(t, testCallAs(t, app, bobTokens.AccessToken, "GET", "/orders/user", "", "", 200))
	if len(bobList["data"].([]any)) != 0 {
		t.Fatalf("other customer's orders leaked: %+v", bobList)
	}
	userList := testBody(t, testCallAs(t, app, tokens.AccessToken, "GET", "/orders/user", "", "", 200))
	if len(userList["data"].([]any)) != 1 {
		t.Fatalf("order listing unexpected: %+v", userList)
	}
	testCall(t, app, "GET", "/orders", "", "", 401).Body.Close()
	testCallAs(t, app, tokens.AccessToken, "GET", "/orders", "", "", 403).Body.Close()
	adminRole := domain.Role{Name: "Admin"}
	if err := tx.Create(&adminRole).Error; err != nil {
		t.Fatal(err)
	}
	admin := domain.User{RoleID: adminRole.ID.ID, Username: "OrdersAdmin", FullName: "Orders Admin", Password: "fixture", IsActive: true}
	if err := tx.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	adminTokens, err := ses.NewTokens(context.Background(), admin.ID.ID, adminRole.ID.ID, "", "Fixture", "127.0.0.3")
	if err != nil {
		t.Fatal(err)
	}
	all := testBody(t, testCallAs(t, app, adminTokens.AccessToken, "GET", "/orders", "", "", 200))
	if len(all["data"].([]any)) != 1 {
		t.Fatalf("admin list failed: %+v", all)
	}
	testCallAs(t, app, adminTokens.AccessToken, "GET", "/orders/"+orderID, "", "", 200).Body.Close()
	var linked int64
	if err := tx.Model(&domain.OrderVoucher{}).Where("order_id = ?", orderID).Count(&linked).Error; err != nil || linked != 2 {
		t.Fatalf("expected both OrderVoucher links, got %d err=%v", linked, err)
	}
}
