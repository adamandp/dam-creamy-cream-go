// Package pricing provides shared active-discount rules and calculations for
// Cart, Catalog, Product Detail and Checkout.
package pricing

import "math"

// ActiveAssociation is the shared SQL predicate against aliases pd and d.
// The storefront must never advertise a discount that Checkout will reject
// because its global redemption quota has already been exhausted.
// Remaining quota is a live snapshot: Checkout still checks and locks the
// discount row for the actual requested quantity during its transaction.
const ActiveAssociation = "pd.is_active = true AND d.start_date <= ? AND d.end_date >= ? AND d.max_redeemed > 0 AND d.redeemed_count < d.max_redeemed"

// DiscountPrice is the price of the PAID product line, not a reward value.
// PRICE is a final price override (bounded by the original price); FIXED is
// an amount subtracted; PERCENTAGE is a percentage subtracted. FREE_ITEM adds
// a separate free product at Checkout and must never lower this line's price.
//
// nil means no line-level price change. PRICE=0 is a valid free price override;
// unlike FIXED=0 or PERCENTAGE=0 it must not be mistaken for "no discount".
// Round to cents here so Cart/Catalog/Detail match Checkout's saved amounts.
func DiscountPrice(price float64, typ string, value *float64) *float64 {
	if typ == "FREE_ITEM" || value == nil || *value < 0 || math.IsNaN(*value) || math.IsInf(*value, 0) {
		return nil
	}
	var final float64
	switch typ {
	case "PRICE":
		final = math.Min(price, *value)
	case "FIXED":
		if *value == 0 {
			return nil
		}
		final = price - *value
	case "PERCENTAGE":
		if *value == 0 {
			return nil
		}
		final = price - price*(*value)/100
	default:
		return nil
	}
	if final < 0 {
		final = 0
	}
	final = math.Round(final*100) / 100
	return &final
}
