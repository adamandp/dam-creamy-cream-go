package vouchers

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
)

type CreateInput struct {
	Code            string             `json:"code"`
	Name            string             `json:"name"`
	Description     *string            `json:"description"`
	Type            string             `json:"type"`
	Group           string             `json:"group"`
	Value           *float64           `json:"value"`
	ProductID       *string            `json:"productId"`
	Quantity        *float64           `json:"quantity"`
	MinimumPurchase float64            `json:"minimumPurchase"`
	IsActive        *bool              `json:"isActive"`
	StartDate       promotionutil.Date `json:"startDate"`
	EndDate         promotionutil.Date `json:"endDate"`
	MaxRedeemed     float64            `json:"maxRedeemed"`
	RedeemedCount   float64            `json:"redeemedCount"`
}
