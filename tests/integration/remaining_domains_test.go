package integration_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/notifications"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/productanalytics"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/recommendations"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/reviews"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/usernotifications"
	"github.com/gofiber/fiber/v3"
)

func TestRemainingFiveNestModulesContracts(t *testing.T) {
	db := connectPrismaFixture(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	create := func(row any) {
		t.Helper()
		if err := tx.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	role := domain.Role{Name: "Customer"}
	create(&role)
	owner := domain.User{RoleID: role.ID.ID, Username: "RemainingOwner", FullName: "Remaining Owner", Password: "fixture", IsActive: true}
	other := domain.User{RoleID: role.ID.ID, Username: "RemainingOther", FullName: "Remaining Other", Password: "fixture", IsActive: true}
	create(&owner)
	create(&other)
	classic := domain.Category{Name: "Classic Collection", IsActive: true}
	otherCategory := domain.Category{Name: "Special Collection", IsActive: true}
	create(&classic)
	create(&otherCategory)
	vanilla := domain.Product{CategoryID: classic.ID.ID, Name: "Vanilla", Price: 20000, IsActive: true}
	chocolate := domain.Product{CategoryID: classic.ID.ID, Name: "Chocolate", Price: 12000, IsActive: true}
	mango := domain.Product{CategoryID: otherCategory.ID.ID, Name: "Mango", Price: 10000, IsActive: true}
	create(&vanilla)
	create(&chocolate)
	create(&mango)
	ses := sessions.NewService(tx, "remaining-access", "remaining-refresh")
	tokens, err := ses.NewTokens(context.Background(), owner.ID.ID, owner.RoleID, "", "Integration", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	reviews.NewService(tx, ses).Routes(app)
	recommendations.NewService(tx).Routes(app)
	productanalytics.NewService(tx, ses).Routes(app)
	notifications.NewService(tx).Routes(app)
	usernotifications.NewService(tx, ses).Routes(app)

	// Reviews preserve source response mappings and global paging totals.
	testBody(t, testCall(t, app, "GET", "/reviews", "", "", 404))
	validReview := fmt.Sprintf(`{"userId":%q,"productId":%q,"rating":5,"comment":"Excellent!"}`, owner.ID.ID, vanilla.ID.ID)
	testBody(t, testCall(t, app, "POST", "/reviews", validReview, "", 201))
	testBody(t, testCall(t, app, "POST", "/reviews", `{"rating":6}`, "", 400))
	var review domain.Review
	if err := tx.First(&review, "product_id = ?", vanilla.ID.ID).Error; err != nil {
		t.Fatal(err)
	}
	create(&domain.Review{UserID: other.ID.ID, ProductID: chocolate.ID.ID, Rating: 4})
	listReviews := requiredRows(t, testBody(t, testCall(t, app, "GET", "/reviews", "", "", 200)))
	if len(listReviews) != 2 {
		t.Fatalf("reviews list: %+v", listReviews)
	}
	productReviews := testBody(t, testCall(t, app, "GET", "/reviews/product/"+vanilla.ID.ID, "", "", 200))
	if productReviews["paging"].(map[string]any)["totalItems"] != float64(2) {
		t.Fatalf("legacy global review count changed: %v", productReviews)
	}
	perProduct := requiredRows(t, productReviews)
	if len(perProduct) != 1 || perProduct[0].(map[string]any)["rate"] != float64(5) || perProduct[0].(map[string]any)["name"] != owner.Username {
		t.Fatalf("review projection differs from source: %v", perProduct)
	}
	testBody(t, testCall(t, app, "GET", "/reviews/"+review.ID.ID, "", "", 200))
	testBody(t, testCall(t, app, "GET", "/reviews/me", "", "", 401))
	mine := testBody(t, testCallAs(t, app, tokens.AccessToken, "GET", "/reviews/me", "", "", 200))
	if len(requiredRows(t, mine)) != 1 || mine["paging"].(map[string]any)["totalItems"] != float64(2) {
		t.Fatalf("reviews/me source paging changed: %v", mine)
	}
	testBody(t, testCall(t, app, "PATCH", "/reviews/"+review.ID.ID, `{"rating":5}`, "", 400))
	testBody(t, testCall(t, app, "PATCH", "/reviews/"+review.ID.ID, `{"rating":3}`, "", 200))

	// Recommendations include active-discount relation, rate and sales ranks.
	discountValue := 25.0
	discount := domain.Discount{Name: "Legacy Offer", DiscountType: "PERCENTAGE", Value: &discountValue, MaxApplied: 3, MaxRedeemed: 10}
	// Prisma Discount requires explicit dates and counts.
	discount.StartDate = review.CreatedAt.Add(-3600000000000)
	discount.EndDate = review.CreatedAt.Add(3600000000000)
	create(&discount)
	create(&domain.ProductDiscount{ProductID: vanilla.ID.ID, DiscountID: discount.ID.ID, IsActive: true})
	classicResponse := testBody(t, testCall(t, app, "GET", "/recommendations/classic", "", "", 200))
	classicRows := requiredRows(t, classicResponse)
	// Earlier PATCH changed Vanilla from five stars to three, so Chocolate
	// (four stars) must now be ranked above it.
	if len(classicRows) != 2 || classicRows[0].(map[string]any)["id"] != chocolate.ID.ID || classicRows[1].(map[string]any)["id"] != vanilla.ID.ID {
		t.Fatalf("classic rating sort: %+v", classicRows)
	}
	if classicRows[1].(map[string]any)["discountPrice"] != float64(15000) {
		t.Fatalf("Nest recommendation discount formula: %+v", classicRows[1])
	}
	categoryRows := requiredRows(t, testBody(t, testCall(t, app, "GET", "/recommendations/categories", "", "", 200)))
	if len(categoryRows) != 2 {
		t.Fatalf("category recommendations: %+v", categoryRows)
	}
	offerRows := requiredRows(t, testBody(t, testCall(t, app, "GET", "/recommendations/offers", "", "", 200)))
	if len(offerRows) != 1 || offerRows[0].(map[string]any)["id"] != vanilla.ID.ID {
		t.Fatalf("offers: %+v", offerRows)
	}
	related := requiredRows(t, testBody(t, testCall(t, app, "GET", "/recommendations/related/"+vanilla.ID.ID, "", "", 200)))
	if len(related) != 2 || related[0].(map[string]any)["id"] == vanilla.ID.ID {
		t.Fatalf("related fallback: %+v", related)
	}
	testBody(t, testCall(t, app, "GET", "/recommendations/related/00000000-0000-4000-8000-000000000010", "", "", 404))

	// Product Analytics: scoped rows with legacy GLOBAL pagination totals.
	analyticsBody := fmt.Sprintf(`{"userId":%q,"productId":%q}`, owner.ID.ID, vanilla.ID.ID)
	testBody(t, testCall(t, app, "POST", "/product-analytics", analyticsBody, "", 201))
	create(&domain.ProductAnalytics{UserID: other.ID.ID, ProductID: chocolate.ID.ID})
	analytics := testBody(t, testCall(t, app, "GET", "/product-analytics/product/"+vanilla.ID.ID, "", "", 200))
	if len(requiredRows(t, analytics)) != 1 || analytics["paging"].(map[string]any)["totalItems"] != float64(2) {
		t.Fatalf("analytics count mismatch: %v", analytics)
	}
	if count := requiredData(t, testBody(t, testCall(t, app, "GET", "/product-analytics/product/"+vanilla.ID.ID+"/count", "", "", 200)))["count"]; count != float64(1) {
		t.Fatalf("analytics product count: %v", count)
	}
	testBody(t, testCall(t, app, "GET", "/product-analytics/user", "", "", 401))
	if rows := requiredRows(t, testBody(t, testCallAs(t, app, tokens.AccessToken, "GET", "/product-analytics/user", "", "", 200))); len(rows) != 1 {
		t.Fatalf("analytics user rows: %+v", rows)
	}

	// Notifications and User Notifications: nested notification JSON and read state.
	testBody(t, testCall(t, app, "GET", "/notifications", "", "", 404))
	testBody(t, testCall(t, app, "POST", "/notifications", `{"title":"Delivery","message":"Your order is ready"}`, "", 201))
	var n domain.Notification
	if err := tx.First(&n, "title = ?", "Delivery").Error; err != nil {
		t.Fatal(err)
	}
	testBody(t, testCall(t, app, "GET", "/notifications", "", "", 200))
	got := requiredData(t, testBody(t, testCall(t, app, "GET", "/notifications/"+n.ID.ID, "", "", 200)))
	if !strings.HasSuffix(got["createdAt"].(string), "Z") {
		t.Fatalf("notification time: %+v", got)
	}
	testBody(t, testCall(t, app, "PATCH", "/notifications/"+n.ID.ID, `{"title":"Delivery"}`, "", 400))
	testBody(t, testCall(t, app, "PATCH", "/notifications/"+n.ID.ID, `{"message":"Shipped"}`, "", 200))
	userNotificationBody := fmt.Sprintf(`{"userId":%q,"notificationId":%q,"isRead":false}`, owner.ID.ID, n.ID.ID)
	testBody(t, testCall(t, app, "POST", "/user-notifications", userNotificationBody, "", 201))
	var un domain.UserNotification
	if err := tx.First(&un, "user_id = ?", owner.ID.ID).Error; err != nil {
		t.Fatal(err)
	}
	userFeed := requiredRows(t, testBody(t, testCall(t, app, "GET", "/user-notifications", "", "", 200)))
	if len(userFeed) != 1 || userFeed[0].(map[string]any)["notifications"].(map[string]any)["message"] != "Shipped" {
		t.Fatalf("nested notification contract: %+v", userFeed)
	}
	testBody(t, testCall(t, app, "GET", "/user-notifications/"+un.ID.ID, "", "", 200))
	testBody(t, testCall(t, app, "GET", "/user-notifications/me", "", "", 401))
	if rows := requiredRows(t, testBody(t, testCallAs(t, app, tokens.AccessToken, "GET", "/user-notifications/me", "", "", 200))); len(rows) != 1 {
		t.Fatalf("user notification feed: %+v", rows)
	}
	count := requiredData(t, testBody(t, testCallAs(t, app, tokens.AccessToken, "GET", "/user-notifications/count/me", "", "", 200)))
	if count["count"] != float64(1) {
		t.Fatalf("notification count: %v", count)
	}
	// Source's notification/:id query checks user_id rather than notification_id.
	oddCount := requiredData(t, testBody(t, testCall(t, app, "GET", "/user-notifications/notification/"+n.ID.ID+"/count", "", "", 200)))
	if oddCount["count"] != float64(0) {
		t.Fatalf("legacy wrong-column count changed: %v", oddCount)
	}
	testBody(t, testCall(t, app, "GET", "/user-notifications/notification/"+n.ID.ID, "", "", 404))
	testBody(t, testCall(t, app, "PATCH", "/user-notifications/"+un.ID.ID, `{}`, "", 200))
	if err := tx.First(&un, "id = ?", un.ID.ID).Error; err != nil || !un.IsRead {
		t.Fatalf("notification read state: %v %+v", err, un)
	}
	if again := requiredData(t, testBody(t, testCallAs(t, app, tokens.AccessToken, "GET", "/user-notifications/count/me", "", "", 200))); again["count"] != float64(1) {
		t.Fatalf("source counts read notifications too: %v", again)
	}
	// Deletion is hard delete, and fails where a UserNotification still references it.
	testBody(t, testCall(t, app, "DELETE", "/notifications/"+n.ID.ID, "", "", 409))
	testBody(t, testCall(t, app, "DELETE", "/reviews/"+review.ID.ID, "", "", 200))
}
