package discounts

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
)

var inputFields = []string{
	"name", "discountType", "value", "maxApplied", "maxRedeemed",
	"redeemedCount", "description", "productId", "quantity", "startDate", "endDate",
}

func validate(b CreateInput) error {
	if err := promotionutil.Name(b.Name, "Discount Name", 1, 100); err != nil {
		return err
	}
	if err := promotionutil.Enum(b.DiscountType, "Discount Type", "PERCENTAGE", "FIXED", "FREE_ITEM", "PRICE"); err != nil {
		return err
	}
	if b.Description != nil {
		if err := promotionutil.Text(*b.Description, "Description", 1, 255); err != nil {
			return err
		}
	}

	return nil
}
