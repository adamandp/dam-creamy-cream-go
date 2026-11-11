package integration_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/addresses"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/carts"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/categories"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/inventory"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/products"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/users"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

func TestInheritedRouteAuthorizationOnPostgres(t *testing.T) {
	db := connectPrismaFixture(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	adminRole := domain.Role{Name: "Admin"}
	customerRole := domain.Role{Name: "Customer"}
	for _, role := range []*domain.Role{&adminRole, &customerRole} {
		if err := tx.Create(role).Error; err != nil {
			t.Fatal(err)
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("OldPassword@123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	admin := domain.User{RoleID: adminRole.ID.ID, Username: "RBACAdmin", FullName: "RBAC Admin", Password: string(hash), IsActive: true}
	alice := domain.User{RoleID: customerRole.ID.ID, Username: "AliceCustomer", FullName: "Alice Customer", Password: string(hash), IsActive: true}
	bob := domain.User{RoleID: customerRole.ID.ID, Username: "BobCustomer", FullName: "Bob Customer", Password: string(hash), IsActive: true}
	for _, u := range []*domain.User{&admin, &alice, &bob} {
		if err := tx.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	ses := sessions.NewService(tx, "rbac-access", "rbac-refresh")
	issue := func(u domain.User) sessions.Tokens {
		tkn, err := ses.NewTokens(context.Background(), u.ID.ID, u.RoleID, "", "Fixture", "127.0.0.1")
		if err != nil {
			t.Fatal(err)
		}
		return tkn
	}
	adminToken, aliceToken, bobToken := issue(admin), issue(alice), issue(bob)
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	fake := stubImageStore{}
	users.NewService(tx, ses, fake).Routes(app)
	addresses.NewService(tx, ses).Routes(app)
	carts.NewService(tx, ses).Routes(app)
	products.NewService(tx, fake, ses).Routes(app)
	inventory.NewService(tx, ses).Routes(app)
	categories.NewService(tx, redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}), ses).Routes(app)
	check := func(token, method, path, body string, want int) {
		t.Helper()
		response := testCallAs(t, app, token, method, path, body, "", want)
		_ = response.Body.Close()
	}
	category := domain.Category{Name: "Current Category", IsActive: true}
	if err := tx.Create(&category).Error; err != nil {
		t.Fatal(err)
	}
	product := domain.Product{CategoryID: category.ID.ID, Name: "Test Vanilla", Price: 12000, IsActive: true}
	if err := tx.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	inventory := domain.Inventory{ProductID: product.ID.ID, Quantity: 11, IsActive: true}
	if err := tx.Create(&inventory).Error; err != nil {
		t.Fatal(err)
	}
	productBody := fmt.Sprintf(`{"categoryId":%q,"name":"Another Vanilla","description":null,"price":12000,"isActive":true}`, category.ID.ID)
	cases := []struct{ method, path, body string }{
		{"POST", "/categories", `{"name":"Ice Cream","description":null,"isActive":true}`},
		{"PATCH", "/categories/" + category.ID.ID, `{"name":"Ice Cream"}`},
		{"DELETE", "/categories/" + category.ID.ID, ""},
		{"POST", "/products", productBody},
		{"PATCH", "/products/" + product.ID.ID, `{"price":15000}`},
		{"PATCH", "/products/" + product.ID.ID + "/image", ""},
		{"DELETE", "/products/" + product.ID.ID, ""},
		{"PATCH", "/inventory/" + inventory.ID.ID, `{"quantity":999}`},
	}
	for _, tc := range cases {
		check("", tc.method, tc.path, tc.body, 401)
		check(aliceToken.AccessToken, tc.method, tc.path, tc.body, 403)
	}
	check(adminToken.AccessToken, "PATCH", "/inventory/"+inventory.ID.ID, `{"quantity":13}`, 200)
	var changed domain.Inventory
	if err := tx.First(&changed, "id = ?", inventory.ID.ID).Error; err != nil || changed.Quantity != 13 {
		t.Fatalf("admin inventory not persisted: %v %+v", err, changed)
	}
	check(adminToken.AccessToken, "PATCH", "/products/"+product.ID.ID, `{"price":15000}`, 200)

	// User directory and per-user PII are never public.
	for _, tc := range []struct {
		path                       string
		anonymous, customer, admin int
	}{
		{"/users", 401, 403, 200},
		{"/users/" + alice.ID.ID, 401, 200, 200},
		{"/users/" + bob.ID.ID, 401, 403, 200},
	} {
		check("", "GET", tc.path, "", tc.anonymous)
		check(aliceToken.AccessToken, "GET", tc.path, "", tc.customer)
		check(adminToken.AccessToken, "GET", tc.path, "", tc.admin)
	}
	// Password change via the generic PATCH endpoint must not bypass the
	// dedicated endpoint's existing-password verification.
	userPath := "/users/" + alice.ID.ID
	for _, token := range []string{aliceToken.AccessToken, adminToken.AccessToken} {
		check(token, "PATCH", userPath, `{"password":"NewPassword@123"}`, 403)
	}
	var unchanged domain.User
	if err := tx.First(&unchanged, "id = ?", alice.ID.ID).Error; err != nil {
		t.Fatal(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(unchanged.Password), []byte("OldPassword@123")) != nil {
		t.Fatal("PATCH unexpectedly modified password")
	}
	check(aliceToken.AccessToken, "PATCH", userPath, `{"fullName":"Updated Alice"}`, 200)

	addrBody := func(userID string) string {
		return fmt.Sprintf(`{"userId":%q,"recipientName":"Jane Doe","recipientPhoneNumber":"081234567890","province":"West Java","city":"Bandung","subdistrict":"Cicendo","village":"Pajajaran","postalCode":"40172","address":"Jalan Merdeka","isPrimary":false}`, userID)
	}
	check("", "POST", "/addresses", addrBody(alice.ID.ID), 401)
	check(bobToken.AccessToken, "POST", "/addresses", addrBody(alice.ID.ID), 403)
	check(aliceToken.AccessToken, "POST", "/addresses", addrBody(bob.ID.ID), 403)
	check(aliceToken.AccessToken, "POST", "/addresses", addrBody(alice.ID.ID), 201)
	var saved domain.Address
	if err := tx.Where("user_id = ?", alice.ID.ID).First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path, body             string
		anonymous, other, owner, admin int
	}{
		{"GET", "/addresses", "", 401, 403, 403, 200},
		{"GET", "/addresses/" + saved.ID.ID, "", 401, 403, 200, 200},
		{"PATCH", "/addresses/" + saved.ID.ID, `{"address":"Jalan Asia"}`, 401, 403, 200, 400},
	} {
		check("", tc.method, tc.path, tc.body, tc.anonymous)
		check(bobToken.AccessToken, tc.method, tc.path, tc.body, tc.other)
		check(aliceToken.AccessToken, tc.method, tc.path, tc.body, tc.owner)
		check(adminToken.AccessToken, tc.method, tc.path, tc.body, tc.admin)
	}
	check(aliceToken.AccessToken, "PATCH", "/addresses/"+saved.ID.ID, fmt.Sprintf(`{"userId":%q}`, bob.ID.ID), 403)
	check(adminToken.AccessToken, "PATCH", "/addresses/"+saved.ID.ID, fmt.Sprintf(`{"userId":%q}`, bob.ID.ID), 403)
	check("", "DELETE", "/addresses/"+saved.ID.ID, "", 401)
	check(bobToken.AccessToken, "DELETE", "/addresses/"+saved.ID.ID, "", 403)
	// Last address cannot be deleted until another exists (legacy invariant).
	check(aliceToken.AccessToken, "DELETE", "/addresses/"+saved.ID.ID, "", 409)
	check(aliceToken.AccessToken, "POST", "/addresses", addrBody(alice.ID.ID), 201)
	check(aliceToken.AccessToken, "DELETE", "/addresses/"+saved.ID.ID, "", 200)

	// A JWT with a valid signature must stop working as a session credential
	// immediately after logout (including on cart and address cookie routes).
	cart := domain.Cart{UserID: alice.ID.ID}
	if err := tx.Create(&cart).Error; err != nil {
		t.Fatal(err)
	}
	cookie := "refreshToken=" + aliceToken.RefreshToken
	for _, route := range []string{"/addresses/user", "/carts/user"} {
		body := testBody(t, testCall(t, app, "GET", route, "", cookie, 200))
		_ = body
	}
	if err := ses.Logout(context.Background(), aliceToken.RefreshToken); err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{"/addresses/user", "/carts/user"} {
		response := testCall(t, app, "GET", route, "", cookie, 401)
		_ = response.Body.Close()
	}
	if _, err := ses.ActiveRefresh(context.Background(), aliceToken.RefreshToken); err == nil {
		t.Fatal("revoked refresh token was accepted")
	}
	// A different current session must continue to be valid.
	if _, err := ses.ActiveRefresh(context.Background(), bobToken.RefreshToken); err != nil {
		t.Fatalf("unrevoked session invalid: %v", err)
	}
	// Session access JWT is intentionally independent and expires within 15m.
}
