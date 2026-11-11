package roles

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func (s *Service) create(c fiber.Ctx) error {
	var body Input
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	if err := httpx.NonBlank(body.Name, 1, 50, "name"); err != nil {
		return err
	}
	if err := s.db.Create(&domain.Role{Name: body.Name}).Error; err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("create", "Role")})
}

func (s *Service) findAll(c fiber.Ctx) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	var data []domain.Role
	var total int64
	if err := s.db.Model(&domain.Role{}).Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return httpx.NotFound("Role")
	}
	if err := s.db.Offset((page - 1) * limit).Limit(limit).Find(&data).Error; err != nil {
		return err
	}
	mapped := make([]view, 0, len(data))
	for _, v := range data {
		mapped = append(mapped, mapRole(v))
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Category"), Data: mapped, Paging: httpx.Pagination(page, limit, total)}) // preserves existing roles.service.ts message
}

func (s *Service) byID(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var role domain.Role
	if err := s.db.First(&role, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("Role")
		}
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Role"), Data: mapRole(role)})
}

func (s *Service) byName(c fiber.Ctx) error {
	var role domain.Role
	if err := s.db.First(&role, "name = ?", c.Params("name")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("Role")
		}
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Role"), Data: mapRole(role)})
}

func (s *Service) update(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var body Input
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	if err := httpx.NonBlank(body.Name, 1, 50, "name"); err != nil {
		return err
	}
	result := s.db.Model(&domain.Role{}).Where("id = ?", id).Update("name", body.Name)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return httpx.NotFound("Role")
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "Role")})
}

func (s *Service) remove(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	result := s.db.Delete(&domain.Role{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return httpx.NotFound("Role")
	}
	return c.JSON(httpx.Response{Message: httpx.Message("delete", "Role")})
}
