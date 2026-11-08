package inventory

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("Inventory")
		}
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
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("Inventory")
		}
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
	var body UpdateInput
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	// Reject invalid quantities without touching persistent state.
	if body.Quantity != nil && (*body.Quantity < 0 || *body.Quantity > 2147483647) {
		return fiber.NewError(400, "quantity must be within the nonnegative 32-bit integer range")
	}
	if body.ReservedQuantity != nil && (*body.ReservedQuantity < 0 || *body.ReservedQuantity > 2147483647) {
		return fiber.NewError(400, "reservedQuantity must be within the nonnegative 32-bit integer range")
	}
	if body.ProductID != nil {
		if err := httpx.UUID(*body.ProductID); err != nil {
			return err
		}
	}
	returnErr := s.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// Serializes stock edits (and future checkout stock reservations)
		// on the same inventory row, so partial updates cannot combine to
		// produce reserved_quantity > quantity.
		var m domain.Inventory
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&m, "id = ?", id).Error; err != nil {
			return err
		}
		changes := map[string]any{}
		if body.ProductID != nil {
			changes["product_id"] = *body.ProductID
		}
		quantity, reserved := m.Quantity, m.ReservedQuantity
		if body.Quantity != nil {
			quantity = *body.Quantity
			changes["quantity"] = quantity
		}
		if body.ReservedQuantity != nil {
			reserved = *body.ReservedQuantity
			changes["reserved_quantity"] = reserved
		}
		if body.IsActive != nil {
			changes["is_active"] = *body.IsActive
		}
		if quantity < 0 || reserved < 0 || reserved > quantity {
			return fiber.NewError(409, "reservedQuantity cannot exceed nonnegative quantity")
		}
		if err := httpx.ValidateUpdate(map[string]any{
			"product_id": m.ProductID, "quantity": m.Quantity,
			"reserved_quantity": m.ReservedQuantity, "is_active": m.IsActive,
		}, changes); err != nil {
			return err
		}
		if len(changes) > 0 {
			return tx.Model(&m).Updates(changes).Error
		}
		return nil
	})
	if returnErr != nil {
		return returnErr
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "Inventory")})
}
