package orders

import (
	"sort"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Service) prepareVouchers(tx *gorm.DB, body CheckoutInput, now time.Time, data *checkoutData) error {
	subtotal := amount(data.totalPrice - data.lineDiscount)
	checkoutCost := subtotal
	shippingCost := data.shippingCost
	originalShippingCost := shippingCost
	seenGroups := map[string]bool{}
	ids := append([]string{}, body.Order.VoucherIDs...)
	sort.Strings(ids)
	for _, id := range ids {
		var v domain.Voucher
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&v, "id = ?", id).Error; err != nil {
			return unavailable("Voucher not found")
		}
		if !v.IsActive || now.Before(v.StartDate) || !now.Before(v.EndDate) {
			return unavailable("Voucher inactive or expired")
		}
		if v.MaxRedeemed <= 0 || v.RedeemedCount >= v.MaxRedeemed {
			return unavailable("Voucher redemption capacity reached")
		}
		if seenGroups[v.Group] {
			return unavailable("Only one voucher per checkout/shipping group")
		}
		if v.Group != "CHECKOUT" && v.Group != "SHIPPING" {
			return unavailable("Unknown voucher group")
		}
		seenGroups[v.Group] = true
		if subtotal < v.MinimumPurchase {
			return unavailable("Voucher minimum purchase not met")
		}
		var claim domain.UserVoucher
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND voucher_id = ?", body.Order.UserID, id).First(&claim).Error; err != nil {
			return unavailable("Voucher was not claimed by this customer")
		}
		if claim.Status != "CLAIMED" || !now.Before(claim.ExpiredAt) {
			return unavailable("Voucher claim is used or expired")
		}
		if v.Type == "FREE_ITEM" {
			if err := addGift(data.allQuantities, &data.gifts, v.ProductID, v.Quantity); err != nil {
				return err
			}
		} else {
			if v.Value == nil || !saneMoney(*v.Value) {
				return unavailable("Voucher value must be nonnegative")
			}
			var source *float64
			if v.Group == "CHECKOUT" {
				source = &checkoutCost
			} else {
				source = &shippingCost
			}
			switch v.Type {
			case "FIXED":
				*source = amount(max(0, *source-*v.Value))
			case "PERCENTAGE":
				if *v.Value > 100 {
					return unavailable("Voucher percent exceeds 100")
				}
				*source = amount(max(0, *source-(*source*(*v.Value)/100)))
			default:
				return unavailable("Unsupported voucher type")
			}
		}
		data.vouchers = append(data.vouchers, claimVoucher{voucher: v, claim: claim})
	}
	data.shippingCost = shippingCost
	// The total must use the same rounded shipping amount that is charged.
	// Mixing the raw request with the cent-rounded value could produce a
	// negative voucher amount even when the buyer applied no voucher.
	data.voucherAmount = amount((subtotal - checkoutCost) + (originalShippingCost - shippingCost))
	data.finalPrice = amount(checkoutCost + shippingCost)
	if !saneMoney(data.voucherAmount) || !saneMoney(data.finalPrice) {
		return unavailable("Invalid voucher-adjusted order total")
	}
	return nil
}
