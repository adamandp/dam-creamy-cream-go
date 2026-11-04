// Package pricing provides shared active-discount rules for Cart and
// product catalog/detail. Eligibility is checked by SQL before calculation.
package pricing

// ActiveAssociation is the shared SQL predicate against aliases pd and d.
const ActiveAssociation = "pd.is_active = true AND d.start_date <= ? AND d.end_date >= ?"

// DiscountPrice uses the same arithmetic for Cart and Catalog. Missing or
// zero-valued discounts have no discount price.
func DiscountPrice(price float64, typ string, value *float64) *float64 {
	if value == nil || *value == 0 {
		return nil
	}
	final := price - *value
	if typ == "PERCENTAGE" {
		final = price - price*(*value)/100
	}
	if final < 0 {
		final = 0
	}
	return &final
}
