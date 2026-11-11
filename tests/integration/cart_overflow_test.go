package integration_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/carts"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
)

func TestCartAtomicUpsertDoesNotOverflowPrismaInteger(t *testing.T) {
	db := connectPrismaFixture(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	role := domain.Role{Name: "OverflowFixture"}
	if err := tx.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := domain.User{RoleID: role.ID.ID, Username: "OverflowFixture", FullName: "Overflow Fixture", Password: "fixture", IsActive: true}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	category := domain.Category{Name: "Overflow Fixture", IsActive: true}
	if err := tx.Create(&category).Error; err != nil {
		t.Fatal(err)
	}
	product := domain.Product{CategoryID: category.ID.ID, Name: "Overflow Vanilla", Price: 3000, IsActive: true}
	if err := tx.Create(&product).Error; err != nil {
		t.Fatal(err)
	}
	cart := domain.Cart{UserID: user.ID.ID}
	if err := tx.Create(&cart).Error; err != nil {
		t.Fatal(err)
	}
	item := domain.CartItem{CartID: cart.ID.ID, ProductID: product.ID.ID, Quantity: 2147483646}
	if err := tx.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	ses := sessions.NewService(tx, "cart-overflow-access", "cart-overflow-refresh")
	tokens, err := ses.NewTokens(context.Background(), user.ID.ID, user.RoleID, "", "Test", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	carts.NewService(tx, ses).Routes(app)
	request := fmt.Sprintf(`{"productId":%q,"quantity":1}`, product.ID.ID)
	testBody(t, testCall(t, app, "POST", "/carts/user/add", request, "refreshToken="+tokens.RefreshToken, 201))
	testBody(t, testCall(t, app, "POST", "/carts/user/add", request, "refreshToken="+tokens.RefreshToken, 409))
	var stored domain.CartItem
	if err := tx.First(&stored, "id = ?", item.ID.ID).Error; err != nil || stored.Quantity != 2147483647 {
		t.Fatalf("cart overflow changed persisted quantity: %+v, err %v", stored, err)
	}
}
