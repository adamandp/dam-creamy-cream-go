package integration_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/discounts"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/productdiscounts"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/uservouchers"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/vouchers"
	"github.com/gofiber/fiber/v3"
)

func requiredData(t *testing.T, result map[string]any) map[string]any {
	t.Helper()
	data, ok := result["data"].(map[string]any)
	if !ok {
		t.Fatalf("missing response data: %+v", result)
	}
	return data
}
func requiredRows(t *testing.T, result map[string]any) []any {
	t.Helper()
	rows, ok := result["data"].([]any)
	if !ok {
		t.Fatalf("response data is not an array: %+v", result)
	}
	return rows
}

func TestPromotionsAPIOnPostgres(t *testing.T) {
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
	user := domain.User{RoleID: role.ID.ID, Username: "PromoFixture", FullName: "Promo Fixture", Password: "test-only", IsActive: true}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	category := domain.Category{Name: "Ice Cream", IsActive: true}
	if err := tx.Create(&category).Error; err != nil {
		t.Fatal(err)
	}
	product := domain.Product{CategoryID: category.ID.ID, Name: "Strawberry", IsActive: true, Price: 25000}
	if err := tx.Create(&product).Error; err != nil {
		t.Fatal(err)
	}

	ses := sessions.NewService(tx, "promo-access-test-secret", "promo-refresh-test-secret")
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	discounts.NewService(tx).Routes(app)
	productdiscounts.NewService(tx).Routes(app)
	vouchers.NewService(tx).Routes(app)
	uservouchers.NewService(tx, ses).Routes(app)

	// The empty Discounts list uses the existing "Users" NotFound label.
	empty := testBody(t, testCall(t, app, "GET", "/discounts", "", "", 404))
	if !strings.Contains(fmt.Sprint(empty["message"]), "Users") {
		t.Fatalf("unexpected Discounts empty response: %+v", empty)
	}
	testBody(t, testCall(t, app, "POST", "/discounts", `{"name":"Winter Deal"}`, "", 400))
	testBody(t, testCall(t, app, "POST", "/discounts", `{"name":"Invalid123","discountType":"FIXED","value":10,"maxApplied":2,"maxRedeemed":100,"redeemedCount":0,"description":null,"productId":null,"quantity":null,"startDate":"2026-01-01","endDate":"2027-01-01"}`, "", 400))
	newDiscount := `{"name":"Winter Deal","discountType":"PERCENTAGE","value":20,"maxApplied":2,"maxRedeemed":100,"redeemedCount":0,"description":null,"productId":null,"quantity":null,"startDate":"2026-01-01T00:00:00.000Z","endDate":"2027-01-01T00:00:00.000Z","ignoreExtra":true}`
	testBody(t, testCall(t, app, "POST", "/discounts", newDiscount, "", 201))
	var d domain.Discount
	if err := tx.First(&d, "name = ?", "Winter Deal").Error; err != nil {
		t.Fatal(err)
	}
	dID := d.ID.ID
	discountRow := requiredData(t, testBody(t, testCall(t, app, "GET", "/discounts/"+dID, "", "", 200)))
	if discountRow["name"] != "Winter Deal" || discountRow["description"] != nil || discountRow["discountType"] != "PERCENTAGE" {
		t.Fatalf("Discount GET contract: %v", discountRow)
	}
	if discountRow["startDate"] != "2026-01-01T00:00:00.000Z" {
		t.Fatalf("Discount date precision: %v", discountRow["startDate"])
	}
	paged := testBody(t, testCall(t, app, "GET", "/discounts?page=1&limit=10", "", "", 200))
	if paged["message"] != httpx.Message("get", "User") || len(requiredRows(t, paged)) != 1 {
		t.Fatalf("Discounts pagination/message contract: %+v", paged)
	}
	testBody(t, testCall(t, app, "PATCH", "/discounts/"+dID, `{"name":"Winter Deal"}`, "", 400))
	testBody(t, testCall(t, app, "PATCH", "/discounts/"+dID, `{"name":"Winter Deal","maxApplied":3}`, "", 400))
	testBody(t, testCall(t, app, "PATCH", "/discounts/"+dID, `{"maxApplied":3,"description":"Limited offer"}`, "", 200))
	testBody(t, testCall(t, app, "PATCH", "/discounts/"+dID, `{"description":null}`, "", 200))
	var changed domain.Discount
	if err := tx.First(&changed, "id = ?", dID).Error; err != nil {
		t.Fatal(err)
	}
	if changed.MaxApplied != 3 || changed.Description != nil {
		t.Fatalf("Discount changes not persisted: %+v", changed)
	}

	// Product-discount association is unique by product_id. Deleting it only
	// switches is_active to false without removing the row.
	badAssociation := fmt.Sprintf(`{"productId":%q,"discountId":"00000000-0000-4000-8000-000000000000"}`, product.ID.ID)
	testBody(t, testCall(t, app, "POST", "/product-discounts", badAssociation, "", 404))
	associationBody := fmt.Sprintf(`{"productId":%q,"discountId":%q}`, product.ID.ID, dID)
	testBody(t, testCall(t, app, "POST", "/product-discounts", associationBody, "", 201))
	var association domain.ProductDiscount
	if err := tx.First(&association, "product_id = ?", product.ID.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !association.IsActive {
		t.Fatal("new association should default to active")
	}
	if err := tx.SavePoint("duplicate_product_discount").Error; err != nil {
		t.Fatal(err)
	}
	testBody(t, testCall(t, app, "POST", "/product-discounts", associationBody, "", 409))
	// Expected PostgreSQL unique violations abort the current transaction until
	// the savepoint is restored; do not poison the rest of this fixture test.
	if err := tx.RollbackTo("duplicate_product_discount").Error; err != nil {
		t.Fatal(err)
	}
	matches := testBody(t, testCall(t, app, "GET", "/product-discounts/product/"+product.ID.ID, "", "", 200))
	if len(requiredRows(t, matches)) != 1 {
		t.Fatalf("product-discount lookup: %v", matches)
	}
	list := testBody(t, testCall(t, app, "GET", "/product-discounts", "", "", 200))
	paging, ok := list["paging"].(map[string]any)
	if !ok || paging["totalItems"] != float64(1) {
		t.Fatalf("public product-discount totalItems must count users: %v", list)
	}
	testBody(t, testCall(t, app, "GET", "/product-discounts/"+association.ID.ID, "", "", 200))
	testBody(t, testCall(t, app, "DELETE", "/product-discounts/"+association.ID.ID, "", "", 200))
	if err := tx.First(&association, "id = ?", association.ID.ID).Error; err != nil || association.IsActive {
		t.Fatalf("product-discount must remain soft-disabled: %v %v", association, err)
	}
	if len(requiredRows(t, testBody(t, testCall(t, app, "GET", "/product-discounts/product/"+product.ID.ID, "", "", 200)))) != 1 {
		t.Fatal("disabled association should remain queryable")
	}

	// Voucher creation preserves required nullable properties and their defaults.
	testBody(t, testCall(t, app, "POST", "/vouchers", `{"code":"WELCOME"}`, "", 400))
	newVoucher := `{"code":"WELCOME20","name":"Welcome Deal","description":null,"type":"PERCENTAGE","group":"CHECKOUT","value":20,"productId":null,"quantity":null,"minimumPurchase":50000,"isActive":true,"startDate":"2026-01-01","endDate":"2027-01-01","maxRedeemed":100,"redeemedCount":0}`
	testBody(t, testCall(t, app, "POST", "/vouchers", newVoucher, "", 201))
	var voucher domain.Voucher
	if err := tx.First(&voucher, "code = ?", "WELCOME20").Error; err != nil {
		t.Fatal(err)
	}
	voucherID := voucher.ID.ID
	voucherData := requiredData(t, testBody(t, testCall(t, app, "GET", "/vouchers/"+voucherID, "", "", 200)))
	if voucherData["code"] != "WELCOME20" || voucherData["group"] != "CHECKOUT" || voucherData["isActive"] != true {
		t.Fatalf("Voucher GET contract: %v", voucherData)
	}
	if len(requiredRows(t, testBody(t, testCall(t, app, "GET", "/vouchers", "", "", 200)))) != 1 {
		t.Fatal("voucher pagination list should contain voucher")
	}
	testBody(t, testCall(t, app, "PATCH", "/vouchers/"+voucherID, `{"name":"Welcome Deal"}`, "", 400))
	testBody(t, testCall(t, app, "PATCH", "/vouchers/"+voucherID, `{"group":"WRONG"}`, "", 400))
	badVoucherProduct := `{"productId":"00000000-0000-4000-8000-000000000000"}`
	testBody(t, testCall(t, app, "PATCH", "/vouchers/"+voucherID, badVoucherProduct, "", 404))
	productLink := fmt.Sprintf(`{"productId":%q}`, product.ID.ID)
	testBody(t, testCall(t, app, "PATCH", "/vouchers/"+voucherID, productLink, "", 200))
	testBody(t, testCall(t, app, "PATCH", "/vouchers/"+voucherID, `{"productId":null,"description":"Weekend only"}`, "", 200))
	testBody(t, testCall(t, app, "DELETE", "/vouchers/"+voucherID, "", "", 200))
	if err := tx.First(&voucher, "id = ?", voucherID).Error; err != nil || voucher.IsActive {
		t.Fatalf("voucher soft-disable should preserve row: %+v err=%v", voucher, err)
	}
	testBody(t, testCall(t, app, "GET", "/vouchers/"+voucherID, "", "", 200))

	// Claims may be created even for a disabled voucher; the API does not
	// reject them or evaluate time-window/redemption limits in this endpoint.
	claimBody := fmt.Sprintf(`{"userId":%q,"voucherId":%q,"claimedAt":"2026-01-01","expiredAt":"2027-01-01"}`, user.ID.ID, voucherID)
	testBody(t, testCall(t, app, "POST", "/user-vouchers", claimBody, "", 201))
	var claim domain.UserVoucher
	if err := tx.First(&claim, "user_id = ?", user.ID.ID).Error; err != nil {
		t.Fatal(err)
	}
	if claim.Status != "CLAIMED" {
		t.Fatalf("new assignment status should be CLAIMED, got %q", claim.Status)
	}
	testBody(t, testCall(t, app, "GET", "/user-vouchers/"+claim.ID.ID, "", "", 200))
	if len(requiredRows(t, testBody(t, testCall(t, app, "GET", "/user-vouchers/voucher/"+voucherID, "", "", 200)))) != 1 {
		t.Fatal("voucher assignments should remain visible")
	}
	if len(requiredRows(t, testBody(t, testCall(t, app, "GET", "/user-vouchers", "", "", 200)))) != 1 {
		t.Fatal("user voucher pagination should include one result")
	}
	testBody(t, testCall(t, app, "GET", "/user-vouchers/user", "", "", 401))
	access, err := ses.NewTokens(context.Background(), user.ID.ID, role.ID.ID, "", "Go test", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("GET", "/user-vouchers/user", nil)
	request.Header.Set("Authorization", "Bearer "+access.AccessToken)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("GET /user-vouchers/user returned %d", response.StatusCode)
	}
	rows := requiredRows(t, testBody(t, response))
	if len(rows) != 1 {
		t.Fatalf("user assignments: %v", rows)
	}
	row := rows[0].(map[string]any)
	nested, ok := row["vouchers"].(map[string]any)
	if !ok || nested["id"] != voucherID || nested["code"] != "WELCOME20" {
		t.Fatalf("nested voucher association missing: %+v", row)
	}
	testBody(t, testCall(t, app, "PATCH", "/user-vouchers/"+claim.ID.ID, "", "", 200))
	testBody(t, testCall(t, app, "PATCH", "/user-vouchers/"+claim.ID.ID, "", "", 200))
	if err := tx.First(&claim, "id = ?", claim.ID.ID).Error; err != nil || claim.Status != "USED" {
		t.Fatalf("assignment must be USED after use: %+v %v", claim, err)
	}

	// Deleting a referenced discount follows the physical FK constraints.
	// Discount removal is a hard delete, unlike voucher and association.
	if err := tx.SavePoint("discount_fk_guard").Error; err != nil {
		t.Fatal(err)
	}
	testBody(t, testCall(t, app, "DELETE", "/discounts/"+dID, "", "", 409))
	if err := tx.RollbackTo("discount_fk_guard").Error; err != nil {
		t.Fatal(err)
	}
	if err := tx.Delete(&association).Error; err != nil {
		t.Fatal(err)
	}
	testBody(t, testCall(t, app, "DELETE", "/discounts/"+dID, "", "", 200))
	testBody(t, testCall(t, app, "GET", "/discounts/"+dID, "", "", 404))
}
