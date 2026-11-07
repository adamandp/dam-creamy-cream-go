package unit_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/modules/orders"
)

func TestCheckoutEstimatedMatchesNestDateCoercion(t *testing.T) {
	expected := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name, value string
		want        time.Time
		invalid     bool
	}{
		{"ISO", `"2026-12-01T00:00:00Z"`, expected, false},
		{"date only", `"2026-12-01"`, expected, false},
		{"epoch milliseconds", `1796083200000`, expected, false},
		{"null coerces to epoch", `null`, time.UnixMilli(0).UTC(), false},
		{"bad date", `"not-a-date"`, time.Time{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := fmt.Sprintf(`{"shipment":{"courier":"jne","service":"regular","cost":8000,"estimated":%s,"description":null}}`, tc.value)
			var request orders.CheckoutInput
			err := json.Unmarshal([]byte(raw), &request)
			if tc.invalid {
				if err == nil {
					t.Fatal("invalid date was accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !request.Shipment.Estimated.Equal(tc.want) {
				t.Fatalf("estimated=%v want=%v", request.Shipment.Estimated, tc.want)
			}
		})
	}
}

func TestCheckoutRequiresShipmentCostAndDescription(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
	}{
		{"missing cost", `{"courier":"jne","service":"regular","estimated":"2026-12-01","description":null}`},
		{"null cost", `{"courier":"jne","service":"regular","cost":null,"estimated":"2026-12-01","description":null}`},
		{"missing description", `{"courier":"jne","service":"regular","cost":0,"estimated":"2026-12-01"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var request orders.CheckoutInput
			if err := json.Unmarshal([]byte(`{"shipment":`+tc.payload+`}`), &request); err == nil {
				t.Fatal("missing required shipment field was accepted")
			}
		})
	}
	var request orders.CheckoutInput
	if err := json.Unmarshal([]byte(`{"shipment":{"courier":"jne","service":"regular","cost":0,"estimated":"2026-12-01","description":null}}`), &request); err != nil {
		t.Fatalf("explicit null description and zero cost should be allowed: %v", err)
	}
}
