package integration_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/addresses"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/auth"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/inventory"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
)

type checkoutAuditFixture struct {
	dbUser  domain.User
	role    domain.Role
	token   string
	refresh string
	app     *fiber.App
}

func newCheckoutAuditFixture(t *testing.T) (*checkoutAuditFixture, func()) {
	t.Helper()
	db := connectPrismaFixture(t)
	role := domain.Role{Name: "Admin"}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := domain.User{RoleID: role.ID.ID, Username: "CheckoutFix" + role.ID.ID, FullName: "Checkout Fix", Password: "fixture-only", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	ses := sessions.NewService(db, "checkoutfix-access", "checkoutfix-refresh")
	tokens, err := ses.NewTokens(context.Background(), user.ID.ID, role.ID.ID, "", "Audit", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	addresses.NewService(db, ses).Routes(app)
	inventory.NewService(db, ses).Routes(app)
	auth.NewService(db, ses, false).Routes(app)
	cleanup := func() {
		_ = db.Where("user_id = ?", user.ID.ID).Delete(&domain.UserToken{}).Error
		_ = db.Where("user_id = ?", user.ID.ID).Delete(&domain.Address{}).Error
		_ = db.Delete(&user).Error
		_ = db.Delete(&role).Error
	}
	return &checkoutAuditFixture{dbUser: user, role: role, token: tokens.AccessToken, refresh: tokens.RefreshToken, app: app}, cleanup
}

func addressPayload(userID, name string, primary bool) string {
	return fmt.Sprintf(`{"userId":%q,"recipientName":%q,"recipientPhoneNumber":"081234567890","province":"West Java","city":"Bandung","subdistrict":"Cicendo","village":"Pajajaran","postalCode":"40172","address":"Jalan Merdeka","isPrimary":%t}`, userID, name, primary)
}

func TestAddressPrimaryInvariantAndReplacement(t *testing.T) {
	db := connectPrismaFixture(t)
	fx, cleanup := newCheckoutAuditFixture(t)
	defer cleanup()
	create := func(name string, primary bool) {
		res := testCallAs(t, fx.app, fx.token, "POST", "/addresses", addressPayload(fx.dbUser.ID.ID, name, primary), "", 201)
		res.Body.Close()
	}
	all := func() []domain.Address {
		var rows []domain.Address
		if err := db.Where("user_id = ?", fx.dbUser.ID.ID).Order("recipient_name").Find(&rows).Error; err != nil {
			t.Fatal(err)
		}
		return rows
	}
	assertPrimary := func(want int) {
		t.Helper()
		n := 0
		for _, a := range all() {
			if a.IsPrimary {
				n++
			}
		}
		if n != want {
			t.Fatalf("expected %d primary, got %d: %+v", want, n, all())
		}
	}
	create("First", false)
	assertPrimary(1)
	create("Second", false)
	assertPrimary(1)
	rows := all()
	var first, second domain.Address
	for _, r := range rows {
		if r.RecipientName == "First" {
			first = r
		} else {
			second = r
		}
	}
	changed := testCallAs(t, fx.app, fx.token, "PATCH", "/addresses/"+second.ID.ID, `{"isPrimary":true}`, "", 200)
	changed.Body.Close()
	assertPrimary(1)
	var actualFirst, actualSecond domain.Address
	_ = db.First(&actualFirst, "id = ?", first.ID.ID).Error
	_ = db.First(&actualSecond, "id = ?", second.ID.ID).Error
	if actualFirst.IsPrimary || !actualSecond.IsPrimary {
		t.Fatal("switch did not demote old primary")
	}
	testCallAs(t, fx.app, fx.token, "PATCH", "/addresses/"+second.ID.ID, `{"isPrimary":false}`, "", 409).Body.Close()
	assertPrimary(1)
	testCallAs(t, fx.app, fx.token, "DELETE", "/addresses/"+second.ID.ID, "", "", 200).Body.Close()
	assertPrimary(1)
	testCallAs(t, fx.app, fx.token, "DELETE", "/addresses/"+first.ID.ID, "", "", 409).Body.Close()
	assertPrimary(1)

	// Two parallel primary requests must serialize on the owner record.
	create("Concurrent A", false)
	create("Concurrent B", false)
	rows = all()
	var targets []string
	for _, a := range rows {
		if strings.HasPrefix(a.RecipientName, "Concurrent") {
			targets = append(targets, a.ID.ID)
		}
	}
	if len(targets) != 2 {
		t.Fatalf("wanted two concurrent candidate addresses: %+v", rows)
	}
	codes := make(chan int, 2)
	var wg sync.WaitGroup
	for _, id := range targets {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			req := httptest.NewRequest("PATCH", "/addresses/"+id, strings.NewReader(`{"isPrimary":true}`))
			req.Header.Set("Authorization", "Bearer "+fx.token)
			req.Header.Set("Content-Type", "application/json")
			resp, err := fx.app.Test(req)
			if err != nil {
				codes <- 0
				return
			}
			resp.Body.Close()
			codes <- resp.StatusCode
		}(id)
	}
	wg.Wait()
	close(codes)
	for code := range codes {
		if code != 200 {
			t.Fatalf("parallel switch rejected: HTTP %d", code)
		}
	}
	assertPrimary(1)
	// Legacy rows may already contain an invalid primary count. A new
	// address should repair both cases without a schema migration.
	if err := db.Model(&domain.Address{}).Where("user_id = ?", fx.dbUser.ID.ID).
		Update("is_primary", false).Error; err != nil {
		t.Fatal(err)
	}
	assertPrimary(0)
	create("Repair Missing Primary", false)
	assertPrimary(1)
	if err := db.Model(&domain.Address{}).Where("user_id = ?", fx.dbUser.ID.ID).
		Update("is_primary", true).Error; err != nil {
		t.Fatal(err)
	}
	create("Repair Multiple Primaries", false)
	assertPrimary(1)
}

func TestInventoryValidationsAndConcurrentEdits(t *testing.T) {
	db := connectPrismaFixture(t)
	fx, cleanup := newCheckoutAuditFixture(t)
	category := domain.Category{Name: "Stock Test Category", IsActive: true}
	if err := db.Create(&category).Error; err != nil {
		t.Fatal(err)
	}
	product := domain.Product{CategoryID: category.ID.ID, Name: "Stock Test Product", Price: 3000, IsActive: true}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	stock := domain.Inventory{ProductID: product.ID.ID, Quantity: 10, ReservedQuantity: 5, IsActive: true}
	if err := db.Create(&stock).Error; err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = db.Delete(&stock).Error
		_ = db.Delete(&product).Error
		_ = db.Delete(&category).Error
		cleanup()
	}()
	path := "/inventory/" + stock.ID.ID
	for _, input := range []struct {
		body   string
		status int
	}{
		{`{"quantity":-4}`, 400},
		{`{"reservedQuantity":-2}`, 400},
		{`{"quantity":2147483648}`, 400},
		{`{"reservedQuantity":2147483648}`, 400},
		{`{"quantity":4}`, 409},
		{`{"reservedQuantity":11}`, 409},
		{`{"quantity":0,"reservedQuantity":0}`, 200},
	} {
		res := testCallAs(t, fx.app, fx.token, "PATCH", path, input.body, "", input.status)
		res.Body.Close()
	}
	testCallAs(t, fx.app, fx.token, "PATCH", path, `{"quantity":10,"reservedQuantity":5}`, "", 200).Body.Close()
	codes := make(chan int, 2)
	var wg sync.WaitGroup
	for _, body := range []string{`{"quantity":6}`, `{"reservedQuantity":8}`} {
		wg.Add(1)
		go func(body string) {
			defer wg.Done()
			req := httptest.NewRequest("PATCH", path, strings.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+fx.token)
			req.Header.Set("Content-Type", "application/json")
			resp, err := fx.app.Test(req)
			if err != nil {
				codes <- 0
				return
			}
			resp.Body.Close()
			codes <- resp.StatusCode
		}(body)
	}
	wg.Wait()
	close(codes)
	success, conflict := 0, 0
	for code := range codes {
		if code == 200 {
			success++
		}
		if code == 409 {
			conflict++
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("concurrent stock update allowed invalid combination: 200=%d 409=%d", success, conflict)
	}
	var stored domain.Inventory
	if err := db.First(&stored, "id = ?", stock.ID.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Quantity < 0 || stored.ReservedQuantity < 0 || stored.ReservedQuantity > stored.Quantity {
		t.Fatalf("stock invariant violated: %+v", stored)
	}
}

func TestRevokedRefreshCannotLogoutAll(t *testing.T) {
	db := connectPrismaFixture(t)
	fx, cleanup := newCheckoutAuditFixture(t)
	defer cleanup()
	ses := sessions.NewService(db, "checkoutfix-access", "checkoutfix-refresh")
	second, err := ses.NewTokens(context.Background(), fx.dbUser.ID.ID, fx.role.ID.ID, "", "Audit Other", "127.0.0.2")
	if err != nil {
		t.Fatal(err)
	}
	if err := ses.Logout(context.Background(), fx.refresh); err != nil {
		t.Fatal(err)
	}
	resp := testCall(t, fx.app, "POST", "/auth/logout-all", "", "refreshToken="+fx.refresh, 401)
	resp.Body.Close()
	if _, err := ses.ActiveRefresh(context.Background(), second.RefreshToken); err != nil {
		t.Fatalf("revoked token deleted a different valid session: %v", err)
	}
	resp = testCall(t, fx.app, "POST", "/auth/logout-all", "", "refreshToken="+second.RefreshToken, 201)
	resp.Body.Close()
	if _, err := ses.ActiveRefresh(context.Background(), second.RefreshToken); err == nil {
		t.Fatal("authorized logout-all did not revoke itself")
	}
	resp = testCall(t, fx.app, "POST", "/auth/logout-all", "", "refreshToken="+second.RefreshToken, 401)
	resp.Body.Close()
}
