// Package legacyviews serializes Prisma-backed domain rows using the existing
// NestJS JSON contract (notably fixed-millisecond UTC DateTime fields).
package legacyviews

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
	"github.com/gofiber/fiber/v3"
)

func Review(m domain.Review) fiber.Map {
	return fiber.Map{
		"id": m.ID.ID, "userId": m.UserID, "productId": m.ProductID,
		"rating": m.Rating, "comment": m.Comment,
		"createdAt": promotionutil.TimeString(m.CreatedAt), "updatedAt": promotionutil.TimeString(m.UpdatedAt),
	}
}

func Analytics(m domain.ProductAnalytics) fiber.Map {
	return fiber.Map{
		"id": m.ID.ID, "userId": m.UserID, "productId": m.ProductID,
		"createdAt": promotionutil.TimeString(m.CreatedAt),
	}
}

func Notification(m domain.Notification) fiber.Map {
	return fiber.Map{
		"id": m.ID.ID, "title": m.Title, "message": m.Message,
		"createdAt": promotionutil.TimeString(m.CreatedAt), "updatedAt": promotionutil.TimeString(m.UpdatedAt),
	}
}

func UserNotification(m domain.UserNotification, related *domain.Notification) fiber.Map {
	view := fiber.Map{
		"id": m.ID.ID, "userId": m.UserID, "notificationId": m.NotificationID,
		"isRead":    m.IsRead,
		"createdAt": promotionutil.TimeString(m.CreatedAt), "updatedAt": promotionutil.TimeString(m.UpdatedAt),
	}
	if related != nil {
		view["notifications"] = Notification(*related)
	}
	return view
}
