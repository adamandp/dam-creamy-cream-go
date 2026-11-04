package unit_test

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"testing"
)

func TestExistingTableMappings(t *testing.T) {
	for _, tc := range []struct{ got, want string }{{(domain.Category{}).TableName(), "category"}, {(domain.UserToken{}).TableName(), "user_tokens"}, {(domain.ProductInformation{}).TableName(), "product_information"}, {(domain.OrderVoucher{}).TableName(), "order_vouchers"}} {
		if tc.got != tc.want {
			t.Errorf("got %s want %s", tc.got, tc.want)
		}
	}
}
