package uservouchers

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
)

type CreateInput struct {
	UserID    string             `json:"userId"`
	VoucherID string             `json:"voucherId"`
	ClaimedAt promotionutil.Date `json:"claimedAt"`
	ExpiredAt promotionutil.Date `json:"expiredAt"`
}
