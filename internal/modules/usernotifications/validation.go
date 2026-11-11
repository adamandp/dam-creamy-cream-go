package usernotifications

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
)

var createFields = []promotionutil.Field{
	{Key: "userId", Label: "User ID", Kind: "uuid"},
	{Key: "notificationId", Label: "Notification ID", Kind: "uuid"},
	{Key: "isRead", Label: "Is Read", Kind: "boolean"},
}
