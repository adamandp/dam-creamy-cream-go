package vouchers

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
)

var inputFields = []string{
	"code", "name", "description", "type", "group", "value", "productId",
	"quantity", "minimumPurchase", "isActive", "startDate", "endDate",
	"maxRedeemed", "redeemedCount",
}

func validate(b CreateInput) error {
	if err := promotionutil.Text(b.Code, "Code", 1, 50); err != nil {
		return err
	}
	if err := promotionutil.Name(b.Name, "Name", 1, 100); err != nil {
		return err
	}
	if b.Description != nil {
		if err := promotionutil.Text(*b.Description, "Description", 1, 255); err != nil {
			return err
		}
	}
	if err := promotionutil.Enum(b.Type, "Discount Type", "PERCENTAGE", "FIXED", "FREE_ITEM"); err != nil {
		return err
	}
	if err := promotionutil.Enum(b.Group, "Group", "CHECKOUT", "SHIPPING"); err != nil {
		return err
	}

	return nil
}
