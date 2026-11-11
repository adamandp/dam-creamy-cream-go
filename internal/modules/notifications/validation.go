package notifications

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
)

var fields = []promotionutil.Field{
	// The database column is VARCHAR(50), even though the source DTO allowed 100.
	{Key: "title", Label: "Title", Kind: "string", Min: 1, Max: 50},
	{Key: "message", Label: "Message", Kind: "string", Min: 1, Max: 255},
}
