package pricing

import "testing"

func TestDiscountPriceSemantics(t *testing.T) {
	tests := []struct {
		name, typ string
		value     *float64
		want      *float64
	}{
		{"percentage", "PERCENTAGE", ptr(20), ptr(8000)},
		{"fractional percentage", "PERCENTAGE", ptr(33.3333), ptr(6666.67)},
		{"fixed", "FIXED", ptr(2000), ptr(8000)},
		{"fixed exceeds base", "FIXED", ptr(15000), ptr(0)},
		{"price override", "PRICE", ptr(7000), ptr(7000)},
		{"zero override", "PRICE", ptr(0), ptr(0)},
		{"override above base", "PRICE", ptr(15000), ptr(10000)},
		{"free item with value", "FREE_ITEM", ptr(7000), nil},
		{"free item no value", "FREE_ITEM", nil, nil},
		{"missing fixed value", "FIXED", nil, nil},
		{"zero fixed", "FIXED", ptr(0), nil},
		{"zero percent", "PERCENTAGE", ptr(0), nil},
		{"unknown type", "UNKNOWN", ptr(500), nil},
		{"negative value", "FIXED", ptr(-500), nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := DiscountPrice(10000, tc.typ, tc.value)
			if (got == nil) != (tc.want == nil) {
				t.Fatalf("nil mismatch: got=%v want=%v", got, tc.want)
			}
			if got != nil && *got != *tc.want {
				t.Fatalf("got=%v want=%v", *got, *tc.want)
			}
		})
	}
}
func ptr(v float64) *float64 { return &v }
