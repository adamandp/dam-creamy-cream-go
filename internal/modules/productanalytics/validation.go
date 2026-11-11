package productanalytics

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
)

var createFields = []promotionutil.Field{
	{Key: "userId", Label: "User id", Kind: "uuid"},
	{Key: "productId", Label: "Product id", Kind: "uuid"},
}
