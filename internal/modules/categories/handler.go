package categories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/gofiber/fiber/v3"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

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
	s.invalidateCache()
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("create", "category")})
}

func (s *Service) findAll(c fiber.Ctx) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	// The revision is shared across application instances and changes after
	// a successful mutation. An old in-flight GET may populate an old key but
	// cannot make newer revisions stale. Old pages expire normally after 1h.
	// Cache is optional: a disconnected Redis must not stall a storefront
	// request (or hide otherwise healthy PostgreSQL data).
	ctx, cancel := context.WithTimeout(c.Context(), 200*time.Millisecond)
	defer cancel()
	revision, cacheErr := s.cache.Get(ctx, categoryRevisionKey).Result()
	if errors.Is(cacheErr, redis.Nil) {
		revision, cacheErr = "0", nil
	}
	key := fmt.Sprintf("categories:v%s:%d:%d", revision, page, limit)
	if cacheErr == nil {
		if value, err := s.cache.Get(ctx, key).Bytes(); err == nil {
			var cached json.RawMessage
			if json.Unmarshal(value, &cached) == nil {
				return c.Type("json").Send(cached)
			}
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
	if cacheErr == nil {
		if b, err := json.Marshal(response); err == nil {
			_ = s.cache.Set(ctx, key, b, time.Hour).Err()
		}
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
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("Category")
		}
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
		s.invalidateCache()
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
	s.invalidateCache()
	return c.JSON(httpx.Response{Message: httpx.Message("delete", "category")})
}
