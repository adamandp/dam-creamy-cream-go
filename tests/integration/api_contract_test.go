package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/addresses"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/auth"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/carts"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/categories"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/inventory"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/products"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/roles"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/users"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

func testCall(t *testing.T, app *fiber.App, method, path, body, cookie string, expected int) *http.Response {
	return testCallAs(t, app, "", method, path, body, cookie, expected)
}
func testCallAs(t *testing.T, app *fiber.App, bearer, method, path, body, cookie string, expected int) *http.Response {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != expected {
		result, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("%s %s returned %d want %d: %s", method, path, resp.StatusCode, expected, string(result))
	}
	return resp
}

func testBody(t *testing.T, r *http.Response) map[string]any {
	t.Helper()
	defer r.Body.Close()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body
}

func TestFirstPassHTTPContractsOnPrismaSchema(t *testing.T) {
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
	hash, err := bcrypt.GenerateFromPassword([]byte("@Password123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	user := domain.User{RoleID: role.ID.ID, Username: "DummyUser", Password: string(hash), FullName: "Dummy User", IsActive: true}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	adminRole := domain.Role{Name: "Admin"}
	if err := tx.Create(&adminRole).Error; err != nil {
		t.Fatal(err)
	}
	admin := domain.User{RoleID: adminRole.ID.ID, Username: "TestAdmin", FullName: "Test Admin", Password: string(hash), IsActive: true}
	if err := tx.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	issued := sessions.NewService(tx, "access-secret-fixture", "refresh-secret-fixture")
	adminTokens, err := issued.NewTokens(context.Background(), admin.ID.ID, adminRole.ID.ID, "", "Integration", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	cart := domain.Cart{UserID: user.ID.ID}
	if err := tx.Create(&cart).Error; err != nil {
		t.Fatal(err)
	}

	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	session := sessions.NewService(tx, "access-secret-fixture", "refresh-secret-fixture")
	fake := stubImageStore{}
	roles.NewService(tx, session).Routes(app)
	categories.NewService(tx, redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})).Routes(app)
	products.NewService(tx, fake).Routes(app)
	users.NewService(tx, session, fake).Routes(app)
	addresses.NewService(tx, session).Routes(app)
	carts.NewService(tx, session).Routes(app)
	inventory.NewService(tx).Routes(app)
	auth.NewService(tx, session, false).Routes(app)
	session.Routes(app)

	// Roles only enforce Zod string length, unlike category baseName.
	testBody(t, testCallAs(t, app, adminTokens.AccessToken, "POST", "/roles", `{"name":"OPS_2026"}`, "", 201))
	testBody(t, testCall(t, app, "POST", "/categories", `{"name":"Ice Cream","description":null}`, "", 400))
	testBody(t, testCall(t, app, "POST", "/categories", `{"name":"123","description":null,"isActive":true}`, "", 400))
	testBody(t, testCall(t, app, "POST", "/categories", `{"name":"Ice Cream","description":null,"isActive":true}`, "", 201))
	var category domain.Category
	if err := tx.Where("name = ?", "Ice Cream").First(&category).Error; err != nil {
		t.Fatal(err)
	}
	invalidProduct := fmt.Sprintf(`{"categoryId":%q,"name":"Vanilla","price":12000,"isActive":true}`, category.ID.ID)
	testBody(t, testCall(t, app, "POST", "/products", invalidProduct, "", 400))
	validProduct := fmt.Sprintf(`{"categoryId":%q,"name":"Vanilla","description":null,"price":12000,"isActive":true}`, category.ID.ID)
	testBody(t, testCall(t, app, "POST", "/products", validProduct, "", 201))
	var product domain.Product
	if err := tx.Where("name = ?", "Vanilla").First(&product).Error; err != nil {
		t.Fatal(err)
	}
	testBody(t, testCall(t, app, "GET", "/products/catalog", "", "", 200))

	var inv = domain.Inventory{ProductID: product.ID.ID, Quantity: 15, IsActive: true}
	if err := tx.Create(&inv).Error; err != nil {
		t.Fatal(err)
	}
	testBody(t, testCall(t, app, "GET", "/inventory/product/"+product.ID.ID, "", "", 200))
	testBody(t, testCall(t, app, "PATCH", "/inventory/"+inv.ID.ID, `{"quantity":20}`, "", 200))

	testBody(t, testCallAs(t, app, adminTokens.AccessToken, "PATCH", "/users/"+user.ID.ID, `{"fullName":"Dummy User"}`, "", 400))
	testBody(t, testCallAs(t, app, adminTokens.AccessToken, "PATCH", "/users/"+user.ID.ID, `{"email":"invalid"}`, "", 400))
	testBody(t, testCallAs(t, app, adminTokens.AccessToken, "PATCH", "/users/"+user.ID.ID, `{"fullName":"Updated User"}`, "", 200))
	testBody(t, testCallAs(t, app, adminTokens.AccessToken, "PATCH", "/users/"+user.ID.ID, `{"gender":"MALE"}`, "", 400))
	testBody(t, testCallAs(t, app, adminTokens.AccessToken, "PATCH", "/users/"+user.ID.ID, `{"gender":"MAN"}`, "", 200))
	var stored domain.User
	if err := tx.First(&stored, "id = ?", user.ID.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Gender == nil || *stored.Gender != "MALE" {
		t.Fatalf("gender should match physical Prisma enum: %+v", stored.Gender)
	}

	login := testCall(t, app, "POST", "/auth/dummy-customer-login", "", "", 201)
	cookie := login.Header.Get("Set-Cookie")
	payload := testBody(t, login)
	if payload["data"] == nil || cookie == "" {
		t.Fatalf("missing access token or refresh cookie: %v", payload)
	}
	testBody(t, testCall(t, app, "POST", "/sessions/refresh", "", cookie, 201))
	unauthorized := testBody(t, testCall(t, app, "GET", "/addresses/user", "", "", 401))
	if unauthorized["message"] != "🚪 You are not logged in. Please sign in to continue! 🔐" {
		t.Errorf("unexpected auth error: %v", unauthorized)
	}
	testBody(t, testCall(t, app, "POST", "/auth/dummy-customer-login", "", cookie, 403))

	addr := fmt.Sprintf(`{"userId":%q,"recipientName":"Jane Doe","recipientPhoneNumber":"081234567890","province":"West Java","city":"Bandung","subdistrict":"Cicendo","village":"Pajajaran","postalCode":"40172","address":"Jalan Merdeka","isPrimary":false}`, user.ID.ID)
	testBody(t, testCall(t, app, "POST", "/addresses", addr, "", 201))
	var primary domain.Address
	if err := tx.Where("user_id = ?", user.ID.ID).First(&primary).Error; err != nil {
		t.Fatal(err)
	}
	if !primary.IsPrimary {
		t.Fatal("first address should be promoted to primary")
	}
	testBody(t, testCall(t, app, "GET", "/addresses/user", "", cookie, 200))

	add := fmt.Sprintf(`{"productId":%q,"quantity":2}`, product.ID.ID)
	testBody(t, testCall(t, app, "POST", "/carts/user/add", add, cookie, 201))
	cartPayload := testBody(t, testCall(t, app, "GET", "/carts/user", "", cookie, 200))
	data, ok := cartPayload["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("cart data differs from Nest contract: %v", cartPayload)
	}
	testBody(t, testCall(t, app, "POST", "/carts/user/remove", add, cookie, 201))
	cartPayload = testBody(t, testCall(t, app, "GET", "/carts/user", "", cookie, 200))
	data, ok = cartPayload["data"].([]any)
	if !ok || len(data) != 0 {
		t.Fatalf("cart removal failed: %v", cartPayload)
	}
	// Nest clears the cookie even when revocation fails.
	invalidLogout := testCall(t, app, "POST", "/auth/logout", "", "refreshToken=not-a-token", 401)
	if !strings.Contains(invalidLogout.Header.Get("Set-Cookie"), "refreshToken=") {
		t.Error("invalid-token logout must clear refresh cookie")
	}
	_ = invalidLogout.Body.Close()
	testBody(t, testCall(t, app, "POST", "/auth/logout", "", cookie, 201))
	testBody(t, testCall(t, app, "POST", "/sessions/refresh", "", cookie, 401))
	newLogin := testCall(t, app, "POST", "/auth/dummy-customer-login", "", "", 201)
	newCookie := newLogin.Header.Get("Set-Cookie")
	testBody(t, newLogin)
	testBody(t, testCall(t, app, "POST", "/auth/logout-all", "", newCookie, 201))
	testBody(t, testCall(t, app, "POST", "/sessions/refresh", "", newCookie, 401))

}

type stubImageStore struct{}

func (stubImageStore) Upload(_ context.Context, _, _ string, _ []byte) (string, error) {
	return "", nil
}
func (stubImageStore) Delete(_ context.Context, _ string) error { return nil }
func (stubImageStore) Get(_ context.Context, _, _, _ string) (infrastructure.StoredImage, error) {
	return infrastructure.StoredImage{}, nil
}
