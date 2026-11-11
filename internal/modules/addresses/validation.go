package addresses

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
)

func validateInput(body CreateInput) error {
	if err := httpx.UUID(body.UserID); err != nil {
		return err
	}
	fields := []struct {
		v, name  string
		min, max int
	}{{body.RecipientName, "recipientName", 1, 100}, {body.RecipientPhoneNumber, "recipientPhoneNumber", 4, 100}, {body.Province, "province", 1, 100}, {body.City, "city", 1, 100}, {body.Subdistrict, "subdistrict", 1, 100}, {body.Village, "village", 1, 100}, {body.PostalCode, "postalCode", 0, 100}, {body.Address, "address", 1, 100}}
	for _, f := range fields {
		if err := httpx.NonBlank(f.v, f.min, f.max, f.name); err != nil {
			return err
		}
	}
	if err := httpx.Name(body.RecipientName, 1, 100, "Recipient Name"); err != nil {
		return err
	}
	if err := httpx.Phone(body.RecipientPhoneNumber); err != nil {
		return err
	}
	return nil
}
