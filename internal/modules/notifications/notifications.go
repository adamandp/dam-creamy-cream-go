package notifications

import (
	"encoding/json"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/legacyviews"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	sessions *sessions.Service
}

func NewService(db *gorm.DB, ses *sessions.Service) *Service {
	return &Service{db: db, sessions: ses}
}

var fields = []promotionutil.Field{
	// The database column is VARCHAR(50), even though the source DTO allowed 100.
	{Key: "title", Label: "Title", Kind: "string", Min: 1, Max: 50},
	{Key: "message", Label: "Message", Kind: "string", Min: 1, Max: 255},
}

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/notifications")
	r.Post("/", s.sessions.RequireAdmin, s.create)
	r.Get("/", s.findAll)
	r.Get("/:id", s.byID)
	r.Patch("/:id", s.sessions.RequireAdmin, s.update)
	r.Delete("/:id", s.sessions.RequireAdmin, s.remove)
}

func (s *Service) create(c fiber.Ctx) error {
	if err := promotionutil.ValidateFields(c, fields, false); err != nil {
		return err
	}
	var body struct {
		Title   string `json:"title"`
		Message string `json:"message"`
	}
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	row := domain.Notification{Title: body.Title, Message: body.Message}
	if err := s.db.WithContext(c.Context()).Create(&row).Error; err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("create", "Notification")})
}

func (s *Service) findAll(c fiber.Ctx) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	var total int64
	if err := s.db.WithContext(c.Context()).Model(&domain.Notification{}).Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return httpx.NotFound("Notification")
	}
	var rows []domain.Notification
	if err := s.db.WithContext(c.Context()).Offset((page - 1) * limit).Limit(limit).Find(&rows).Error; err != nil {
		return err
	}
	data := make([]fiber.Map, 0, len(rows))
	for _, row := range rows {
		data = append(data, legacyviews.Notification(row))
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Notification"), Data: data, Paging: httpx.Pagination(page, limit, total)})
}

func (s *Service) byID(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var row domain.Notification
	if err := s.db.WithContext(c.Context()).First(&row, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("Notification")
		}
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Notification"), Data: legacyviews.Notification(row)})
}

func (s *Service) update(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var row domain.Notification
	if err := s.db.WithContext(c.Context()).First(&row, "id = ?", id).Error; err != nil {
		return err
	}
	if err := promotionutil.ValidateFields(c, fields, true); err != nil {
		return err
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(c.Body(), &body); err != nil {
		return err
	}
	changes := map[string]any{}
	for _, field := range []string{"title", "message"} {
		if raw, ok := body[field]; ok {
			var value string
			if err := json.Unmarshal(raw, &value); err != nil {
				return err
			}
			changes[field] = value
		}
	}
	if err := promotionutil.ValidateUpdateFields(map[string]any{"title": row.Title, "message": row.Message}, changes, fields...); err != nil {
		return err
	}
	if err := s.db.WithContext(c.Context()).Model(&row).Updates(changes).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "Notification")})
}

func (s *Service) remove(c fiber.Ctx) error {
	var row domain.Notification
	if err := s.db.WithContext(c.Context()).First(&row, "id = ?", c.Params("id")).Error; err != nil {
		return err
	}
	if err := s.db.WithContext(c.Context()).Delete(&row).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("delete", "Notification")})
}
