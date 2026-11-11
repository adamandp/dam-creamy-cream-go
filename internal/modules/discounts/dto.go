package discounts

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
)

type CreateInput struct {
	Name          string             `json:"name"`
	DiscountType  string             `json:"discountType"`
	Value         *float64           `json:"value"`
	MaxApplied    float64            `json:"maxApplied"`
	MaxRedeemed   float64            `json:"maxRedeemed"`
	RedeemedCount float64            `json:"redeemedCount"`
	Description   *string            `json:"description"`
	ProductID     *string            `json:"productId"`
	Quantity      *float64           `json:"quantity"`
	StartDate     promotionutil.Date `json:"startDate"`
	EndDate       promotionutil.Date `json:"endDate"`
}
