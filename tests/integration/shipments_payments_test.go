package integration_test

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/payments"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/shipments"
	"github.com/gofiber/fiber/v3"
)

func TestShipmentOwnershipAdminStatusAndPendingPaymentScaffold(t *testing.T) {
	db := connectPrismaFixture(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	customerRole := domain.Role{Name: "Customer"}
	adminRole := domain.Role{Name: "Admin"}
	for _, role := range []*domain.Role{&customerRole, &adminRole} {
		if err := tx.Create(role).Error; err != nil {
			t.Fatal(err)
		}
	}
	owner := domain.User{RoleID: customerRole.ID.ID, Username: "ShippingOwner", FullName: "Shipping Owner", Password: "fixture", IsActive: true}
	other := domain.User{RoleID: customerRole.ID.ID, Username: "ShippingOther", FullName: "Shipping Other", Password: "fixture", IsActive: true}
	admin := domain.User{RoleID: adminRole.ID.ID, Username: "ShippingAdmin", FullName: "Shipping Admin", Password: "fixture", IsActive: true}
	for _, u := range []*domain.User{&owner, &other, &admin} {
		if err := tx.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	address := domain.Address{
		UserID: owner.ID.ID, RecipientName: "Owner", RecipientPhoneNumber: "081234567890",
		Province: "West Java", City: "Bandung", Subdistrict: "Cicendo",
		Village: "Pajajaran", PostalCode: "40172", Address: "Jalan Merdeka",
	}
	if err := tx.Create(&address).Error; err != nil {
		t.Fatal(err)
	}
	order := domain.Order{UserID: owner.ID.ID, AddressID: address.ID.ID, TotalPrice: 10000, FinalPrice: 15000, Status: "PENDING"}
	if err := tx.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	shipment := domain.Shipment{
		OrderID: order.ID.ID, Courier: "jne", Service: "REG", Cost: 5000,
		Estimated: time.Now().UTC().Truncate(time.Second).Add(48 * time.Hour), Status: "PENDING",
	}
	if err := tx.Create(&shipment).Error; err != nil {
		t.Fatal(err)
	}
	ses := sessions.NewService(tx, "shipments-access", "shipments-refresh")
	issue := func(u domain.User) string {
		t.Helper()
		issued, err := ses.NewTokens(context.Background(), u.ID.ID, u.RoleID, "", "Fixture", "127.0.0.1")
		if err != nil {
			t.Fatal(err)
		}
		return issued.AccessToken
	}
	ownerToken := issue(owner)
	otherToken := issue(other)
	adminToken := issue(admin)
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	shipments.NewService(tx, ses, nil, nil).Routes(app)
	payments.NewService().Routes(app)

	testBody(t, testCall(t, app, "GET", "/shipments", "", "", 401))
	testBody(t, testCallAs(t, app, ownerToken, "GET", "/shipments", "", "", 403))
	feed := testBody(t, testCallAs(t, app, adminToken, "GET", "/shipments?page=1&limit=10", "", "", 200))
	if feed["paging"] == nil || len(feed["data"].([]any)) != 1 {
		t.Fatalf("shipment list missing data/paging: %v", feed)
	}
	testBody(t, testCall(t, app, "GET", "/shipments/"+shipment.ID.ID, "", "", 401))
	testBody(t, testCallAs(t, app, otherToken, "GET", "/shipments/"+shipment.ID.ID, "", "", 403))
	found := testBody(t, testCallAs(t, app, ownerToken, "GET", "/shipments/"+shipment.ID.ID, "", "", 200))
	row := found["data"].(map[string]any)
	if estimated, ok := row["estimated"].(string); !ok || !strings.HasSuffix(estimated, ".000Z") {
		t.Fatalf("Prisma estimated timestamp mismatch: %v", row["estimated"])
	}
	prismaTimestamp := regexp.MustCompile(`\.\d{3}Z$`)
	for _, field := range []string{"createdAt", "updatedAt"} {
		if date, ok := row[field].(string); !ok || !prismaTimestamp.MatchString(date) {
			t.Fatalf("Prisma %s timestamp mismatch: %v", field, row[field])
		}
	}
	listed := feed["data"].([]any)[0].(map[string]any)
	if estimated, ok := listed["estimated"].(string); !ok || !strings.HasSuffix(estimated, ".000Z") {
		t.Fatalf("Prisma list timestamp mismatch: %v", listed["estimated"])
	}
	if row["id"] != shipment.ID.ID {
		t.Fatalf("shipment lookup leaked wrong data: %v", found)
	}
	testBody(t, testCallAs(t, app, adminToken, "GET", "/shipments/"+shipment.ID.ID, "", "", 200))
	testBody(t, testCallAs(t, app, adminToken, "GET", "/shipments/invalid-uuid", "", "", 400))

	url := "/shipments/" + shipment.ID.ID + "/status"
	testBody(t, testCall(t, app, "PATCH", url, `{"status":"SHIPPED"}`, "", 401))
	testBody(t, testCallAs(t, app, ownerToken, "PATCH", url, `{"status":"SHIPPED"}`, "", 403))
	testBody(t, testCallAs(t, app, adminToken, "PATCH", url, `{"status":"INVALID"}`, "", 400))
	testBody(t, testCallAs(t, app, adminToken, "PATCH", url, `{}`, "", 400))
	testBody(t, testCallAs(t, app, adminToken, "PATCH", url, `{"status":"SHIPPED","legacyExtra":"ignored"}`, "", 200))
	if err := tx.First(&shipment, "id = ?", shipment.ID.ID).Error; err != nil || shipment.Status != "SHIPPED" {
		t.Fatalf("shipment status not persisted: %v %+v", err, shipment)
	}
	if err := tx.First(&order, "id = ?", order.ID.ID).Error; err != nil || order.Status != "PENDING" {
		t.Fatalf("shipment PATCH altered order state: %v %+v", err, order)
	}
	testBody(t, testCall(t, app, "POST", "/shipments/cost", `{"origin":1,"destination":2,"weight":100}`, "", 503))
	var before, after int64
	if err := tx.Model(&domain.Payment{}).Count(&before).Error; err != nil {
		t.Fatal(err)
	}
	testCall(t, app, "POST", "/payments", `{}`, "", 201).Body.Close()
	if err := tx.Model(&domain.Payment{}).Count(&after).Error; err != nil || before != after {
		t.Fatalf("legacy payment placeholder persisted a charge: %v before=%d after=%d", err, before, after)
	}
}
