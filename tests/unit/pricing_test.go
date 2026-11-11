package unit_test

import (
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/modules/pricing"
)

func TestDiscountPriceSemantics(t *testing.T) {
	tests := []struct {
		name, typ string
		value     *float64
		want      *float64
	}{
		{"percentage", "PERCENTAGE", floatPtr(20), floatPtr(8000)},
		{"fractional percentage", "PERCENTAGE", floatPtr(33.3333), floatPtr(6666.67)},
		{"fixed", "FIXED", floatPtr(2000), floatPtr(8000)},
		{"fixed exceeds base", "FIXED", floatPtr(15000), floatPtr(0)},
		{"price override", "PRICE", floatPtr(7000), floatPtr(7000)},
		{"zero override", "PRICE", floatPtr(0), floatPtr(0)},
		{"override above base", "PRICE", floatPtr(15000), floatPtr(10000)},
		{"free item with value", "FREE_ITEM", floatPtr(7000), nil},
		{"free item no value", "FREE_ITEM", nil, nil},
		{"missing fixed value", "FIXED", nil, nil},
		{"zero fixed", "FIXED", floatPtr(0), nil},
		{"zero percent", "PERCENTAGE", floatPtr(0), nil},
		{"unknown type", "UNKNOWN", floatPtr(500), nil},
		{"negative value", "FIXED", floatPtr(-500), nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := pricing.DiscountPrice(10000, tc.typ, tc.value)
			if (got == nil) != (tc.want == nil) {
				t.Fatalf("nil mismatch: got=%v want=%v", got, tc.want)
			}
			if got != nil && *got != *tc.want {
				t.Fatalf("got=%v want=%v", *got, *tc.want)
			}
		})
	}
}
func floatPtr(v float64) *float64 { return &v }
