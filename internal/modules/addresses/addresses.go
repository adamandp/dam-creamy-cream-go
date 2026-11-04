package addresses

import (
	"context"

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

func NewService(db *gorm.DB, s *sessions.Service) *Service { return &Service{db: db, sessions: s} }

type CreateInput struct {
	UserID               string `json:"userId"`
	RecipientName        string `json:"recipientName"`
	RecipientPhoneNumber string `json:"recipientPhoneNumber"`
	Province             string `json:"province"`
	City                 string `json:"city"`
	Subdistrict          string `json:"subdistrict"`
	Village              string `json:"village"`
	PostalCode           string `json:"postalCode"`
	Address              string `json:"address"`
	IsPrimary            bool   `json:"isPrimary"`
}
type UpdateInput struct {
	UserID               *string `json:"userId"`
	RecipientName        *string `json:"recipientName"`
	RecipientPhoneNumber *string `json:"recipientPhoneNumber"`
	Province             *string `json:"province"`
	City                 *string `json:"city"`
	Subdistrict          *string `json:"subdistrict"`
	Village              *string `json:"village"`
	PostalCode           *string `json:"postalCode"`
	Address              *string `json:"address"`
	IsPrimary            *bool   `json:"isPrimary"`
}

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/addresses")
	r.Post("/", s.sessions.RequireActor, s.create)
	r.Get("/", s.sessions.RequireAdmin, s.findAll)
	r.Get("/user", s.findByUser)
	r.Get("/:id", s.sessions.RequireActor, s.findOne)
	r.Patch("/:id", s.sessions.RequireActor, s.update)
	r.Delete("/:id", s.sessions.RequireActor, s.remove)
}
func validateInput(body CreateInput) error {
	if err := httpx.UUID(body.UserID); err != nil {
		return err
	}
	fields := []struct {
		v, name  string
		min, max int
	}{{body.RecipientName, "recipientName", 1, 100}, {body.RecipientPhoneNumber, "recipientPhoneNumber", 4, 100}, {body.Province, "province", 1, 100}, {body.City, "city", 1, 100}, {body.Subdistrict, "subdistrict", 1, 100}, {body.Village, "village", 1, 100}, {body.PostalCode, "postalCode", 0, 100}, {body.Address, "address", 1, 100}}
	for _, f := range fields {
		if err := httpx.NonBlank(f.v, f.min, f.max, f.name); err != nil {
			return err
		}
	}
	if err := httpx.Name(body.RecipientName, 1, 100, "Recipient Name"); err != nil {
		return err
	}
	if err := httpx.Phone(body.RecipientPhoneNumber); err != nil {
		return err
	}
	return nil
}
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
	err = s.db.WithContext(context.Background()).Transaction(func(tx *gorm.DB) error {
		var userCount int64
		if err := tx.Model(&domain.User{}).Where("id = ?", body.UserID).Count(&userCount).Error; err != nil {
			return err
		}
		if userCount == 0 {
			return httpx.NotFound("User")
		}
		var count int64
		if err := tx.Model(&domain.Address{}).Where("user_id = ?", body.UserID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			body.IsPrimary = true
		}
		if body.IsPrimary {
			if err := tx.Model(&domain.Address{}).Where("user_id = ? AND is_primary = true", body.UserID).Update("is_primary", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(&domain.Address{UserID: body.UserID, RecipientName: body.RecipientName, RecipientPhoneNumber: body.RecipientPhoneNumber, Province: body.Province, City: body.City, Subdistrict: body.Subdistrict, Village: body.Village, PostalCode: body.PostalCode, Address: body.Address, IsPrimary: body.IsPrimary}).Error
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
	if err := httpx.ValidateUpdate(map[string]any{"user_id": m.UserID, "recipient_name": m.RecipientName, "recipient_phone_number": m.RecipientPhoneNumber, "province": m.Province, "city": m.City, "subdistrict": m.Subdistrict, "village": m.Village, "postal_code": m.PostalCode, "address": m.Address, "is_primary": m.IsPrimary}, values); err != nil {
		return err
	}
	if len(values) > 0 {
		if err := s.db.Model(&m).Updates(values).Error; err != nil {
			return err
		}
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "Address")})
}
func (s *Service) remove(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var m domain.Address
		if err := tx.First(&m, "id = ?", id).Error; err != nil {
			return err
		}
		actor, err := s.sessions.Actor(c)
		if err != nil {
			return err
		}
		if !actor.Admin && actor.UserID != m.UserID {
			return httpx.Forbidden()
		}
		var count int64
		if err := tx.Model(&domain.Address{}).Where("user_id = ?", m.UserID).Count(&count).Error; err != nil {
			return err
		}
		if count <= 1 {
			return fiber.NewError(409, "❌ Cannot delete Address! At least one Address is required to keep the system functional. 🔒")
		}
		return tx.Delete(&m).Error
	})
	if err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("delete", "Address")})
}
