package orders

import (
	"strings"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
)

func TestFreeItemQuantityMatchesNestWithoutArbitraryHundredCap(t *testing.T) {
	productID, quantity := "reward-product", 101
	counts := map[string]int{}
	var gifts []domain.OrderItem
	if err := addGift(counts, &gifts, &productID, &quantity); err != nil {
		t.Fatalf("valid 101-unit reward was rejected: %v", err)
	}
	if counts[productID] != 101 || len(gifts) != 1 || gifts[0].Quantity != 101 {
		t.Fatalf("reward quantities incorrect: %+v %+v", counts, gifts)
	}
	quantity = 0
	if err := addGift(counts, &gifts, &productID, &quantity); err == nil {
		t.Fatal("zero reward quantity should be rejected")
	}
	quantity = 2147483647
	if err := addGift(counts, &gifts, &productID, &quantity); err == nil {
		t.Fatal("aggregate reward quantity overflow should be rejected")
	}
}

func TestCheckoutDescriptionAndCourierMatchNestUTF16Limits(t *testing.T) {
	const userID = "123e4567-e89b-42d3-a456-426614174000"
	const addressID = "123e4567-e89b-42d3-a456-426614174001"
	const productID = "123e4567-e89b-42d3-a456-426614174002"
	var request CheckoutInput
	request.Order.UserID = userID
	request.Order.AddressID = addressID
	request.OrderItems = []CheckoutItem{{ProductID: productID, Quantity: 1}}
	request.Shipment.Cost = 1000
	request.Shipment.Estimated = time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	request.Shipment.Courier = strings.Repeat("中", 50)
	request.Shipment.Service = "regular"
	request.Payment.Method = "cash"
	description := strings.Repeat("中", 255)
	request.Shipment.Description = &description
	actor := sessions.Actor{UserID: userID}
	if err := validateCheckout(request, actor); err != nil {
		t.Fatalf("NestJS permits 255 CJK characters: %v", err)
	}
	description = strings.Repeat("🍦", 128) // 256 UTF-16 units
	if err := validateCheckout(request, actor); err == nil {
		t.Fatal("NestJS rejects a description exceeding 255 UTF-16 units")
	}
}
