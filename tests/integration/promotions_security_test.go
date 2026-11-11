package integration_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/discounts"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/productdiscounts"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/roles"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/users"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/uservouchers"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/vouchers"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func fixtureActor(t *testing.T, db *gorm.DB, name, roleName string) (domain.User, domain.Role) {
	t.Helper()
	role := domain.Role{Name: roleName}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := domain.User{RoleID: role.ID.ID, Username: name, FullName: name, Password: "test-only", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	return user, role
}

func fixtureBearer(t *testing.T, ses *sessions.Service, user domain.User) string {
	t.Helper()
	v, err := ses.NewTokens(context.Background(), user.ID.ID, user.RoleID, "", "Audit", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	return v.AccessToken
}

func auditPost(t *testing.T, app *fiber.App, token, method, path, body string) int {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	return res.StatusCode
}

func auditStatus(t *testing.T, app *fiber.App, token, method, path, body string, expected int) {
	t.Helper()
	actual := auditPost(t, app, token, method, path, body)
	if actual != expected {
		t.Fatalf("%s %s status %d expected %d (authenticated=%t)", method, path, actual, expected, token != "")
	}
}

func TestAdminAuthorizationAndNoRoleEscalation(t *testing.T) {
	db := connectPrismaFixture(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	admin, adminRole := fixtureActor(t, tx, "SecureAdmin", "Admin")
	customer, customerRole := fixtureActor(t, tx, "SecureCustomer", "Customer")
	ses := sessions.NewService(tx, "secure-access", "secure-refresh")
	adminJWT := fixtureBearer(t, ses, admin)
	userJWT := fixtureBearer(t, ses, customer)
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	roles.NewService(tx, ses).Routes(app)
	users.NewService(tx, ses, stubImageStore{}).Routes(app)
	discounts.NewService(tx, ses).Routes(app)
	productdiscounts.NewService(tx, ses).Routes(app)
	vouchers.NewService(tx, ses).Routes(app)
	uservouchers.NewService(tx, ses).Routes(app)
	_ = adminRole

	rolePath := "/roles/" + customerRole.ID.ID
	// An anonymous caller or normal customer cannot rename their role to Admin.
	auditStatus(t, app, "", "PATCH", rolePath, `{"name":"Admin"}`, 401)
	auditStatus(t, app, userJWT, "PATCH", rolePath, `{"name":"Admin"}`, 403)
	auditStatus(t, app, userJWT, "GET", "/user-vouchers", "", 403)
	auditStatus(t, app, "", "POST", "/roles", `{"name":"HiddenAdmin"}`, 401)
	auditStatus(t, app, userJWT, "POST", "/roles", `{"name":"HiddenAdmin"}`, 403)
	auditStatus(t, app, adminJWT, "POST", "/roles", `{"name":"Auditor"}`, 201)
	var auditor domain.Role
	if err := tx.First(&auditor, "name = ?", "Auditor").Error; err != nil {
		t.Fatal(err)
	}
	auditStatus(t, app, adminJWT, "PATCH", "/roles/"+auditor.ID.ID, `{"name":"AuditorUpdated"}`, 200)
	auditStatus(t, app, adminJWT, "DELETE", "/roles/"+auditor.ID.ID, "", 200)
	auditStatus(t, app, "", "DELETE", rolePath, "", 401)
	auditStatus(t, app, userJWT, "DELETE", rolePath, "", 403)

	// Current customer can edit their own profile but must not set roleId or isActive.
	userPath := "/users/" + customer.ID.ID
	auditStatus(t, app, "", "PATCH", userPath, `{"roleId":"`+adminRole.ID.ID+`"}`, 401)
	auditStatus(t, app, userJWT, "PATCH", userPath, fmt.Sprintf(`{"roleId":%q}`, adminRole.ID.ID), 403)
	auditStatus(t, app, userJWT, "PATCH", userPath, `{"isActive":false}`, 403)
	auditStatus(t, app, userJWT, "PATCH", userPath, `{"fullName":"New Customer"}`, 200)
	auditStatus(t, app, userJWT, "DELETE", userPath, "", 403)
	auditStatus(t, app, "", "PUT", userPath+"/password", `{"exsitingPassword":"a","newPassword":"b"}`, 401)
	auditStatus(t, app, adminJWT, "PUT", userPath+"/password", `{"exsitingPassword":"a","newPassword":"b"}`, 403)
	auditStatus(t, app, "", "DELETE", userPath, "", 401)

	promoBody := `{"code":"SECURE77","name":"Secure Voucher","description":null,"type":"FIXED","group":"CHECKOUT","value":10,"productId":null,"quantity":null,"minimumPurchase":0,"isActive":true,"startDate":"2020-01-01","endDate":"2099-01-01","maxRedeemed":10,"redeemedCount":0}`
	discountBody := `{"name":"Secure Deal","discountType":"FIXED","value":10,"maxApplied":1,"maxRedeemed":10,"redeemedCount":0,"description":null,"productId":null,"quantity":null,"startDate":"2020-01-01","endDate":"2099-01-01"}`
	for _, tc := range []struct{ path, body string }{
		{"/vouchers", promoBody}, {"/discounts", discountBody},
		{"/product-discounts", `{}`},
	} {
		auditStatus(t, app, "", "POST", tc.path, tc.body, 401)
		auditStatus(t, app, userJWT, "POST", tc.path, tc.body, 403)
		auditStatus(t, app, "not-a-valid-token", "POST", tc.path, tc.body, 401)
	}
	// Association soft-deletion is a write too; no customer bypass.
	for _, tokenAndStatus := range []struct {
		token  string
		status int
	}{{"", 401}, {userJWT, 403}} {
		auditStatus(t, app, tokenAndStatus.token, "DELETE", "/product-discounts/00000000-0000-4000-8000-000000000000", "", tokenAndStatus.status)
	}
	auditStatus(t, app, adminJWT, "POST", "/vouchers", promoBody, 201)
	auditStatus(t, app, adminJWT, "POST", "/discounts", discountBody, 201)
	var voucher domain.Voucher
	if err := tx.First(&voucher, "code = ?", "SECURE77").Error; err != nil {
		t.Fatal(err)
	}
	var discount domain.Discount
	if err := tx.First(&discount, "name = ?", "Secure Deal").Error; err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/vouchers/" + voucher.ID.ID, "/discounts/" + discount.ID.ID} {
		auditStatus(t, app, "", "PATCH", path, `{"maxRedeemed":8}`, 401)
		auditStatus(t, app, userJWT, "PATCH", path, `{"maxRedeemed":8}`, 403)
		auditStatus(t, app, "", "DELETE", path, "", 401)
		auditStatus(t, app, userJWT, "DELETE", path, "", 403)
	}
	auditStatus(t, app, adminJWT, "PATCH", "/vouchers/"+voucher.ID.ID, `{"maxRedeemed":8}`, 200)
	auditStatus(t, app, adminJWT, "PATCH", "/discounts/"+discount.ID.ID, `{"maxRedeemed":8}`, 200)

	// Admin-only role assignment immediately revokes a JWT whose role claim
	// does not match the new database role. No stale privilege is retained.
	auditStatus(t, app, adminJWT, "PATCH", userPath, fmt.Sprintf(`{"roleId":%q}`, adminRole.ID.ID), 200)
	auditStatus(t, app, userJWT, "GET", "/user-vouchers", "", 401)
}

func TestClaimVoucherRulesAndConcurrentLimit(t *testing.T) {
	// Use a real pool and committed rows, not a single shared GORM transaction;
	// otherwise concurrent requests cannot exercise PostgreSQL FOR UPDATE.
	db := connectPrismaFixture(t)
	a, role := fixtureActor(t, db, "ClaimConcurrencyA", "Customer")
	b := domain.User{RoleID: role.ID.ID, Username: "ClaimConcurrencyB", FullName: "Claim Concurrency B", Password: "test-only", IsActive: true}
	if err := db.Create(&b).Error; err != nil {
		t.Fatal(err)
	}
	admin, adminRole := fixtureActor(t, db, "ClaimConcurrencyAdmin", "Admin")
	now := time.Now().UTC()
	voucher := domain.Voucher{Code: "CAPACITY1", Name: "Capacity One", Type: "FIXED", Group: "CHECKOUT", MinimumPurchase: 0, IsActive: true, StartDate: now.Add(-time.Hour), EndDate: now.Add(24 * time.Hour), MaxRedeemed: 1}
	if err := db.Create(&voucher).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// Clean up test-only rows in dependency order.
		_ = db.Where("user_id IN ?", []string{a.ID.ID, b.ID.ID, admin.ID.ID}).Delete(&domain.UserToken{}).Error
		_ = db.Where("voucher_id = ?", voucher.ID.ID).Delete(&domain.UserVoucher{}).Error
		_ = db.Delete(&voucher).Error
		_ = db.Delete(&a).Error
		_ = db.Delete(&b).Error
		_ = db.Delete(&admin).Error
		_ = db.Delete(&role).Error
		_ = db.Delete(&adminRole).Error
	})
	ses := sessions.NewService(db, "claims-access", "claims-refresh")
	aToken := fixtureBearer(t, ses, a)
	bToken := fixtureBearer(t, ses, b)
	adminToken := fixtureBearer(t, ses, admin)
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	uservouchers.NewService(db, ses).Routes(app)
	claim := func(id string) string {
		return fmt.Sprintf(`{"voucherId":%q,"userId":%q,"claimedAt":"2020-01-01","expiredAt":"2099-01-01"}`, voucher.ID.ID, id)
	}
	result := make(chan int, 2)
	var start sync.WaitGroup
	start.Add(2)
	for _, item := range []struct{ jwt, id string }{{aToken, a.ID.ID}, {bToken, b.ID.ID}} {
		go func(jwt, id string) {
			defer start.Done()
			req := httptest.NewRequest("POST", "/user-vouchers", strings.NewReader(claim(id)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+jwt)
			resp, err := app.Test(req)
			if err != nil {
				result <- 0
				return
			}
			_ = resp.Body.Close()
			result <- resp.StatusCode
		}(item.jwt, item.id)
	}
	start.Wait()
	close(result)
	success, conflict := 0, 0
	for code := range result {
		if code == 201 {
			success++
		}
		if code == 409 {
			conflict++
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("concurrent capacity violation: success=%d conflict=%d", success, conflict)
	}
	var claims []domain.UserVoucher
	if err := db.Where("voucher_id = ?", voucher.ID.ID).Find(&claims).Error; err != nil {
		t.Fatal(err)
	}
	if len(claims) != 1 {
		t.Fatalf("expected one claim, got %d", len(claims))
	}
	if claims[0].ClaimedAt.Before(now.Add(-2*time.Millisecond)) || absDuration(claims[0].ExpiredAt.Sub(voucher.EndDate)) > 2*time.Millisecond {
		t.Fatalf("client-controlled claim dates must not be stored: %+v", claims[0])
	}
	if claims[0].UserID == a.ID.ID {
		auditStatus(t, app, aToken, "POST", "/user-vouchers", claim(a.ID.ID), 409)
	}
	if claims[0].UserID == b.ID.ID {
		auditStatus(t, app, bToken, "POST", "/user-vouchers", claim(b.ID.ID), 409)
	}
	auditStatus(t, app, adminToken, "GET", "/user-vouchers", "", 200)

	// Reject inactive, expired, not-yet-started and already exhausted vouchers.
	for _, tc := range []struct {
		code       string
		active     bool
		start, end time.Time
		redeemed   int
	}{
		{"OFF", false, now.Add(-time.Hour), now.Add(time.Hour), 0},
		{"EXPIRED", true, now.Add(-48 * time.Hour), now.Add(-24 * time.Hour), 0},
		{"FUTURE", true, now.Add(time.Hour), now.Add(2 * time.Hour), 0},
		{"USEDUP", true, now.Add(-time.Hour), now.Add(time.Hour), 1},
	} {
		v := domain.Voucher{Code: tc.code, Name: "Limited Voucher", Type: "FIXED", Group: "CHECKOUT", IsActive: tc.active, StartDate: tc.start, EndDate: tc.end, MaxRedeemed: 1, RedeemedCount: tc.redeemed}
		if err := db.Create(&v).Error; err != nil {
			t.Fatal(err)
		}
		payload := fmt.Sprintf(`{"voucherId":%q}`, v.ID.ID)
		auditStatus(t, app, aToken, "POST", "/user-vouchers", payload, 409)
		_ = db.Delete(&v).Error
	}
	// A minimal voucherId-only payload is enough for the customer claim flow;
	// userId and dates are derived server-side. Never rely on client expiry.
	plain := domain.Voucher{Code: "MINIMALCLAIM", Name: "Minimal Claim", Type: "FIXED", Group: "CHECKOUT", IsActive: true, StartDate: now.Add(-time.Hour), EndDate: now.Add(12 * time.Hour), MaxRedeemed: 5}
	if err := db.Create(&plain).Error; err != nil {
		t.Fatal(err)
	}
	defer db.Delete(&plain)
	minimal := fmt.Sprintf(`{"voucherId":%q}`, plain.ID.ID)
	auditStatus(t, app, aToken, "POST", "/user-vouchers", minimal, 201)
	auditStatus(t, app, aToken, "POST", "/user-vouchers", minimal, 409)
	var stored domain.UserVoucher
	if err := db.Where("user_id = ? AND voucher_id = ?", a.ID.ID, plain.ID.ID).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.UserID != a.ID.ID || stored.ClaimedAt.Before(now.Add(-time.Second)) || absDuration(stored.ExpiredAt.Sub(plain.EndDate)) > 2*time.Millisecond {
		t.Fatalf("server-owned minimal claim: %+v", stored)
	}
	if err := db.Delete(&stored).Error; err != nil {
		t.Fatal(err)
	}
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
