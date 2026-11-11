package addresses

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Service) create(c fiber.Ctx) error {
	var body CreateInput
	if err := httpx.BindStrict(c, &body); err != nil {
		return err
	}
	if err := httpx.Required(c, "userId", "recipientName", "recipientPhoneNumber", "province", "city", "subdistrict", "village", "postalCode", "address", "isPrimary"); err != nil {
		return err
	}
	if httpx.ExplicitNull(c, "postalCode") {
		return fiber.NewError(400, "postalCode must be a string")
	}
	if err := validateInput(body); err != nil {
		return err
	}
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	if !actor.Admin && actor.UserID != body.UserID {
		return httpx.Forbidden()
	}
	err = s.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		// Lock per user, not per address: concurrent creates and primary
		// switches serialize even before the user's first address exists.
		var user domain.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&user, "id = ?", body.UserID).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&domain.Address{}).Where("user_id = ?", body.UserID).Count(&count).Error; err != nil {
			return err
		}
		// Repair pre-existing zero/multiple-primary records from the old
		// implementation when a new address is added.
		var existingPrimaries int64
		if count > 0 {
			if err := tx.Model(&domain.Address{}).
				Where("user_id = ? AND is_primary = true", body.UserID).
				Count(&existingPrimaries).Error; err != nil {
				return err
			}
		}
		makePrimary := body.IsPrimary || existingPrimaries != 1
		if makePrimary {
			if err := tx.Model(&domain.Address{}).
				Where("user_id = ? AND is_primary = true", body.UserID).
				Update("is_primary", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(&domain.Address{
			UserID: body.UserID, RecipientName: body.RecipientName,
			RecipientPhoneNumber: body.RecipientPhoneNumber, Province: body.Province,
			City: body.City, Subdistrict: body.Subdistrict, Village: body.Village,
			PostalCode: body.PostalCode, Address: body.Address, IsPrimary: makePrimary,
		}).Error
	})
	if err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{StatusCode: 201, Message: httpx.Message("create", "Address")})
}

func (s *Service) findAll(c fiber.Ctx) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	var data []domain.Address
	var total int64
	if err := s.db.Model(&domain.Address{}).Count(&total).Error; err != nil {
		return err
	}
	if err := s.db.Offset((page - 1) * limit).Limit(limit).Find(&data).Error; err != nil {
		return err
	}
	if data == nil {
		data = []domain.Address{}
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Address"), Data: data, Paging: httpx.Pagination(page, limit, total)})
}

func (s *Service) findByUser(c fiber.Ctx) error {
	claims, err := s.sessions.CookieUser(c)
	if err != nil {
		return err
	}
	var items []domain.Address
	if err := s.db.Where("user_id = ?", claims.Subject).Find(&items).Error; err != nil {
		return err
	}
	if items == nil {
		items = []domain.Address{}
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "User Address"), Data: items})
}

func (s *Service) findOne(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var m domain.Address
	if err := s.db.First(&m, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("Address")
		}
		return err
	}
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	if !actor.Admin && actor.UserID != m.UserID {
		return httpx.Forbidden()
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Address"), Data: m})
}

func (s *Service) update(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var m domain.Address
	if err := s.db.First(&m, "id = ?", id).Error; err != nil {
		return err
	}
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	if !actor.Admin && actor.UserID != m.UserID {
		return httpx.Forbidden()
	}
	var body UpdateInput
	if err := httpx.BindStrict(c, &body); err != nil {
		return err
	}
	if err := httpx.RejectNull(c, "userId", "recipientName", "recipientPhoneNumber", "province", "city", "subdistrict", "village", "postalCode", "address", "isPrimary"); err != nil {
		return err
	}
	values := map[string]any{}
	fields := []struct {
		v         *string
		name, col string
		min, max  int
	}{{body.RecipientName, "recipientName", "recipient_name", 1, 100}, {body.RecipientPhoneNumber, "recipientPhoneNumber", "recipient_phone_number", 4, 100}, {body.Province, "province", "province", 1, 100}, {body.City, "city", "city", 1, 100}, {body.Subdistrict, "subdistrict", "subdistrict", 1, 100}, {body.Village, "village", "village", 1, 100}, {body.PostalCode, "postalCode", "postal_code", 0, 100}, {body.Address, "address", "address", 1, 100}}
	for _, f := range fields {
		if f.v != nil {
			if err := httpx.NonBlank(*f.v, f.min, f.max, f.name); err != nil {
				return err
			}
			values[f.col] = *f.v
		}
	}
	// Address ownership must not be transferred through a generic PATCH.
	// Admin may manage address fields, but user reassignment is not supported.
	if body.UserID != nil {
		return httpx.Forbidden()
	}
	if body.IsPrimary != nil {
		values["is_primary"] = *body.IsPrimary
	}
	if body.RecipientName != nil {
		if err := httpx.Name(*body.RecipientName, 1, 100, "Recipient Name"); err != nil {
			return err
		}
	}
	if body.RecipientPhoneNumber != nil {
		if err := httpx.Phone(*body.RecipientPhoneNumber); err != nil {
			return err
		}
	}
	// Serializing all address writes through the owner row prevents two
	// concurrent primary switches from committing as two primary addresses.
	err = s.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		var owner domain.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&owner, "id = ?", m.UserID).Error; err != nil {
			return err
		}
		var current domain.Address
		if err := tx.First(&current, "id = ?", id).Error; err != nil {
			return err
		}
		// Requests may have queued behind an earlier delete; never change
		// ownership or rely on a stale snapshot.
		if current.UserID != m.UserID {
			return httpx.Forbidden()
		}
		if body.IsPrimary != nil && !*body.IsPrimary && current.IsPrimary {
			return fiber.NewError(409, "Choose another primary address before unsetting the current one")
		}
		if err := httpx.ValidateUpdate(map[string]any{
			"user_id": current.UserID, "recipient_name": current.RecipientName,
			"recipient_phone_number": current.RecipientPhoneNumber,
			"province":               current.Province, "city": current.City,
			"subdistrict": current.Subdistrict, "village": current.Village,
			"postal_code": current.PostalCode, "address": current.Address,
			"is_primary": current.IsPrimary,
		}, values); err != nil {
			return err
		}
		if len(values) == 0 {
			return nil
		}
		if body.IsPrimary != nil && *body.IsPrimary {
			if err := tx.Model(&domain.Address{}).
				Where("user_id = ? AND id <> ? AND is_primary = true", current.UserID, id).
				Update("is_primary", false).Error; err != nil {
				return err
			}
		}
		return tx.Model(&current).Updates(values).Error
	})
	if err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "Address")})
}

func (s *Service) remove(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	// Read only to find which user row to lock. The target is re-read
	// inside the transaction and rechecked for ownership.
	var target domain.Address
	if err := s.db.First(&target, "id = ?", id).Error; err != nil {
		return err
	}
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	if !actor.Admin && actor.UserID != target.UserID {
		return httpx.Forbidden()
	}
	err = s.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		var owner domain.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&owner, "id = ?", target.UserID).Error; err != nil {
			return err
		}
		var current domain.Address
		if err := tx.First(&current, "id = ?", id).Error; err != nil {
			return err
		}
		if current.UserID != target.UserID {
			return httpx.Forbidden()
		}
		var count int64
		if err := tx.Model(&domain.Address{}).
			Where("user_id = ?", current.UserID).Count(&count).Error; err != nil {
			return err
		}
		if count <= 1 {
			return fiber.NewError(409, "❌ Cannot delete Address! At least one Address is required to keep the system functional. 🔒")
		}
		// When deleting the primary, deterministically nominate a survivor.
		// Both operations commit together and cannot race another address write.
		if current.IsPrimary {
			var next domain.Address
			if err := tx.Where("user_id = ? AND id <> ?", current.UserID, id).
				Order("id ASC").First(&next).Error; err != nil {
				return err
			}
			if err := tx.Model(&domain.Address{}).
				Where("user_id = ? AND id <> ? AND is_primary = true", current.UserID, next.ID.ID).
				Update("is_primary", false).Error; err != nil {
				return err
			}
			if err := tx.Model(&next).Update("is_primary", true).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&current).Error
	})
	if err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("delete", "Address")})
}
