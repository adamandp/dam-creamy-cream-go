package categories

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	cache    *redis.Client
	sessions *sessions.Service
}

func NewService(db *gorm.DB, cache *redis.Client, ses *sessions.Service) *Service {
	return &Service{db: db, cache: cache, sessions: ses}
}

type CreateInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"isActive"`
}
type UpdateInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"isActive"`
}

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/categories")
	r.Post("/", s.sessions.RequireAdmin, s.create)
	r.Get("/", s.findAll)
	r.Get("/:id", s.findOne)
	r.Patch("/:id", s.sessions.RequireAdmin, s.update)
	r.Delete("/:id", s.sessions.RequireAdmin, s.remove)
}
func (s *Service) create(c fiber.Ctx) error {
	var body CreateInput
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	if err := httpx.Required(c, "name", "description", "isActive"); err != nil {
		return err
	}
	if body.IsActive == nil {
		return fiber.NewError(400, "isActive must be a boolean")
	}
	if err := httpx.Name(body.Name, 1, 100, "name"); err != nil {
		return err
	}
	if body.Description != nil {
		if err := httpx.NonBlank(*body.Description, 1, 255, "description"); err != nil {
			return err
		}
	}
	m := domain.Category{Name: body.Name, Description: body.Description, IsActive: *body.IsActive}
	if err := s.db.Create(&m).Error; err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("create", "category")})
}
func (s *Service) findAll(c fiber.Ctx) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("categories:%d:%d", page, limit)
	if value, err := s.cache.Get(context.Background(), key).Bytes(); err == nil {
		var cached json.RawMessage
		if json.Unmarshal(value, &cached) == nil {
			return c.Type("json").Send(cached)
		}
	}
	var data []domain.Category
	var total int64
	if err := s.db.Model(&domain.Category{}).Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return httpx.NotFound("category")
	}
	if err := s.db.Offset((page - 1) * limit).Limit(limit).Find(&data).Error; err != nil {
		return err
	}
	response := httpx.Response{Message: httpx.Message("get", "category"), Data: data, Paging: httpx.Pagination(page, limit, total)}
	if b, err := json.Marshal(response); err == nil {
		_ = s.cache.Set(context.Background(), key, b, time.Hour).Err()
	}
	return c.JSON(response)
}
func (s *Service) findOne(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var m domain.Category
	if err := s.db.First(&m, "id = ?", id).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "category"), Data: m})
}
func (s *Service) update(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var m domain.Category
	if err := s.db.First(&m, "id = ?", id).Error; err != nil {
		return err
	}
	var body UpdateInput
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	if err := httpx.RejectNull(c, "name", "isActive"); err != nil {
		return err
	}
	changes := map[string]any{}
	if body.Name != nil {
		if err := httpx.Name(*body.Name, 1, 100, "name"); err != nil {
			return err
		}
		changes["name"] = *body.Name
	}
	if body.Description != nil {
		if err := httpx.NonBlank(*body.Description, 1, 255, "description"); err != nil {
			return err
		}
		changes["description"] = *body.Description
	}
	if body.IsActive != nil {
		changes["is_active"] = *body.IsActive
	}
	if httpx.ExplicitNull(c, "description") {
		changes["description"] = nil
	}
	if err := httpx.ValidateUpdate(map[string]any{"name": m.Name, "description": valueOrNil(m.Description), "is_active": m.IsActive}, changes); err != nil {
		return err
	}
	if len(changes) > 0 {
		if err := s.db.Model(&m).Updates(changes).Error; err != nil {
			return err
		}
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "category")})
}
func (s *Service) remove(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	result := s.db.Delete(&domain.Category{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return httpx.NotFound("category")
	}
	return c.JSON(httpx.Response{Message: httpx.Message("delete", "category")})
}

func valueOrNil(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}
