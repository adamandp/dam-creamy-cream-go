package usernotifications

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/legacyviews"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func (s *Service) create(c fiber.Ctx) error {
	if err := promotionutil.ValidateFields(c, createFields, false); err != nil {
		return err
	}
	var in struct {
		UserID         string `json:"userId"`
		NotificationID string `json:"notificationId"`
		IsRead         bool   `json:"isRead"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	var user domain.User
	if err := s.db.WithContext(c.Context()).First(&user, "id = ?", in.UserID).Error; err != nil {
		return err
	}
	var notification domain.Notification
	if err := s.db.WithContext(c.Context()).First(&notification, "id = ?", in.NotificationID).Error; err != nil {
		return err
	}
	m := domain.UserNotification{UserID: in.UserID, NotificationID: in.NotificationID, IsRead: in.IsRead}
	if err := s.db.WithContext(c.Context()).Create(&m).Error; err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("create", "User Notification")})
}

func (s *Service) withNotifications(c fiber.Ctx, rows []domain.UserNotification) ([]fiber.Map, error) {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.NotificationID)
	}
	related := map[string]domain.Notification{}
	if len(ids) > 0 {
		var notifications []domain.Notification
		if err := s.db.WithContext(c.Context()).Where("id IN ?", ids).Find(&notifications).Error; err != nil {
			return nil, err
		}
		for _, notification := range notifications {
			related[notification.ID.ID] = notification
		}
	}
	data := make([]fiber.Map, 0, len(rows))
	for _, row := range rows {
		notification, exists := related[row.NotificationID]
		if !exists {
			data = append(data, legacyviews.UserNotification(row, nil))
			continue
		}
		data = append(data, legacyviews.UserNotification(row, &notification))
	}
	return data, nil
}

func (s *Service) list(c fiber.Ctx, filter string, argument any, scopedCount bool) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	query := s.db.WithContext(c.Context()).Model(&domain.UserNotification{})
	if filter != "" {
		query = query.Where(filter, argument)
	}
	var rows []domain.UserNotification
	if err := query.Offset((page - 1) * limit).Limit(limit).Find(&rows).Error; err != nil {
		return err
	}
	var total int64
	count := s.db.WithContext(c.Context()).Model(&domain.UserNotification{})
	if scopedCount && filter != "" {
		count = count.Where(filter, argument)
	}
	if err := count.Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return httpx.NotFound("User Notification")
	}
	data, err := s.withNotifications(c, rows)
	if err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "User Notification"), Data: data, Paging: httpx.Pagination(page, limit, total)})
}

func (s *Service) findAll(c fiber.Ctx) error { return s.list(c, "", nil, false) }

func (s *Service) byID(c fiber.Ctx) error {
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var row domain.UserNotification
	if err := s.db.WithContext(c.Context()).First(&row, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("User Notification")
		}
		return err
	}
	if !actor.Admin && actor.UserID != row.UserID {
		return httpx.Forbidden()
	}
	views, err := s.withNotifications(c, []domain.UserNotification{row})
	if err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "User Notification"), Data: views[0]})
}

func (s *Service) byUser(c fiber.Ctx) error {
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	return s.list(c, "user_id = ?", actor.UserID, true)
}

func (s *Service) userCount(c fiber.Ctx) error {
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	var total int64
	if err := s.db.WithContext(c.Context()).Model(&domain.UserNotification{}).Where("user_id = ?", actor.UserID).Count(&total).Error; err != nil {
		return err
	}
	// NestJS counts every user notification, not only unread notifications.
	return c.JSON(httpx.Response{Message: httpx.Message("get", "User Notification"), Data: fiber.Map{"count": total}})
}

func (s *Service) ensureNotification(c fiber.Ctx) (string, error) {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return "", err
	}
	var n domain.Notification
	if err := s.db.WithContext(c.Context()).First(&n, "id = ?", id).Error; err != nil {
		return "", err
	}
	return id, nil
}

func (s *Service) byNotification(c fiber.Ctx) error {
	id, err := s.ensureNotification(c)
	if err != nil {
		return err
	}
	return s.list(c, "notification_id = ?", id, true)
}

func (s *Service) notificationCount(c fiber.Ctx) error {
	id, err := s.ensureNotification(c)
	if err != nil {
		return err
	}
	var total int64
	if err := s.db.WithContext(c.Context()).Model(&domain.UserNotification{}).Where("notification_id = ?", id).Count(&total).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "User Notification"), Data: fiber.Map{"count": total}})
}

func (s *Service) updateRead(c fiber.Ctx) error {
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	var row domain.UserNotification
	if err := s.db.WithContext(c.Context()).First(&row, "id = ?", c.Params("id")).Error; err != nil {
		return err
	}
	if !actor.Admin && actor.UserID != row.UserID {
		return httpx.Forbidden()
	}
	if err := s.db.WithContext(c.Context()).Model(&row).Update("is_read", true).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "User Notification")})
}
