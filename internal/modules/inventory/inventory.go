package inventory

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	sessions *sessions.Service
}

func NewService(db *gorm.DB, ses *sessions.Service) *Service { return &Service{db: db, sessions: ses} }
func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/inventory")
	r.Get("/", s.findAll)
	r.Get("/product/:id", s.byProduct)
	r.Get("/:id", s.byID)
	r.Patch("/:id", s.sessions.RequireAdmin, s.update)
}
func (s *Service) findAll(c fiber.Ctx) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	var data []domain.Inventory
	var total int64
	if err := s.db.Model(&domain.Inventory{}).Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return httpx.NotFound("Inventory")
	}
	if err := s.db.Offset((page - 1) * limit).Limit(limit).Find(&data).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Inventory"), Data: data, Paging: httpx.Pagination(page, limit, total)})
}
func (s *Service) byID(c fiber.Ctx) error {
	if err := httpx.UUID(c.Params("id")); err != nil {
		return err
	}
	var m domain.Inventory
	if err := s.db.First(&m, "id = ?", c.Params("id")).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Inventory"), Data: m})
}
func (s *Service) byProduct(c fiber.Ctx) error {
	if err := httpx.UUID(c.Params("id")); err != nil {
		return err
	}
	var m domain.Inventory
	if err := s.db.First(&m, "product_id = ?", c.Params("id")).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Inventory"), Data: m})
}

type UpdateInput struct {
	ProductID        *string `json:"productId"`
	Quantity         *int    `json:"quantity"`
	ReservedQuantity *int    `json:"reservedQuantity"`
	IsActive         *bool   `json:"isActive"`
}

func (s *Service) update(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var m domain.Inventory
	if err := s.db.First(&m, "id = ?", id).Error; err != nil {
		return err
	}
	var body UpdateInput
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	changes := map[string]any{}
	if body.ProductID != nil {
		if err := httpx.UUID(*body.ProductID); err != nil {
			return err
		}
		changes["product_id"] = *body.ProductID
	}
	if body.Quantity != nil {
		changes["quantity"] = *body.Quantity
	}
	if body.ReservedQuantity != nil {
		changes["reserved_quantity"] = *body.ReservedQuantity
	}
	if body.IsActive != nil {
		changes["is_active"] = *body.IsActive
	}
	if err := httpx.ValidateUpdate(map[string]any{
		"product_id": m.ProductID, "quantity": m.Quantity, "reserved_quantity": m.ReservedQuantity, "is_active": m.IsActive,
	}, changes); err != nil {
		return err
	}
	if len(changes) > 0 {
		if err := s.db.Model(&m).Updates(changes).Error; err != nil {
			return err
		}
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "Inventory")})
}
