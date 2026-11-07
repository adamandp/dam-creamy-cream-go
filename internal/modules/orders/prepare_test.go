package orders

import (
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
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
