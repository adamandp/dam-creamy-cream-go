package recommendations

import (
	"sort"
)

func productViews(rows []recommendation) []recommendationView {
	views := make([]recommendationView, 0, len(rows))
	for _, row := range rows {
		views = append(views, row.view())
	}
	return views
}

func rankByRatingAndSales(rows []recommendation) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Rate != rows[j].Rate {
			return rows[i].Rate > rows[j].Rate
		}
		return rows[i].Sales > rows[j].Sales
	})
}

// decorate uses bulk queries so the NestJS relations (reviews, order items,
// active ProductDiscount and its Discount) produce the same response fields.
