package vouchers

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
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
	r := app.Group("/vouchers")
	r.Post("/", s.sessions.RequireAdmin, s.create)
	r.Get("/", s.findAll)
	r.Get("/:id", s.byID)
	r.Patch("/:id", s.sessions.RequireAdmin, s.update)
	r.Delete("/:id", s.sessions.RequireAdmin, s.remove)
}

var inputFields = []string{
	"code", "name", "description", "type", "group", "value", "productId",
	"quantity", "minimumPurchase", "isActive", "startDate", "endDate",
	"maxRedeemed", "redeemedCount",
}

type CreateInput struct {
	Code            string             `json:"code"`
	Name            string             `json:"name"`
	Description     *string            `json:"description"`
	Type            string             `json:"type"`
	Group           string             `json:"group"`
	Value           *float64           `json:"value"`
	ProductID       *string            `json:"productId"`
	Quantity        *float64           `json:"quantity"`
	MinimumPurchase float64            `json:"minimumPurchase"`
	IsActive        *bool              `json:"isActive"`
	StartDate       promotionutil.Date `json:"startDate"`
	EndDate         promotionutil.Date `json:"endDate"`
	MaxRedeemed     float64            `json:"maxRedeemed"`
	RedeemedCount   float64            `json:"redeemedCount"`
}

func validate(b CreateInput) error {
	if err := promotionutil.Text(b.Code, "Code", 1, 50); err != nil {
		return err
	}
	if err := promotionutil.Name(b.Name, "Name", 1, 100); err != nil {
		return err
	}
	if b.Description != nil {
		if err := promotionutil.Text(*b.Description, "Description", 1, 255); err != nil {
			return err
		}
	}
	if err := promotionutil.Enum(b.Type, "Discount Type", "PERCENTAGE", "FIXED", "FREE_ITEM"); err != nil {
		return err
	}
	if err := promotionutil.Enum(b.Group, "Group", "CHECKOUT", "SHIPPING"); err != nil {
		return err
	}

	return nil
}

func (s *Service) create(c fiber.Ctx) error {
	if err := promotionutil.ValidateFields(c, promotionutil.VoucherFields(), false); err != nil {
		return err
	}
	var body CreateInput
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	if err := promotionutil.Required(c, inputFields...); err != nil {
		return err
	}
	if err := httpx.RejectNull(c, "code", "name", "type", "group", "minimumPurchase", "isActive", "maxRedeemed", "redeemedCount"); err != nil {
		return err
	}
	if body.IsActive == nil {
		return fiber.NewError(400, "isActive must be a boolean")
	}
	if err := validate(body); err != nil {
		return err
	}
	m := domain.Voucher{
		Code: body.Code, Name: body.Name, Description: body.Description, Type: body.Type,
		Group: body.Group, Value: body.Value, ProductID: body.ProductID,
		Quantity: promotionutil.NullableInteger(body.Quantity), MinimumPurchase: body.MinimumPurchase,
		IsActive: *body.IsActive, StartDate: body.StartDate.Time, EndDate: body.EndDate.Time,
		MaxRedeemed: promotionutil.Integer(body.MaxRedeemed), RedeemedCount: promotionutil.Integer(body.RedeemedCount),
	}
	if err := s.db.Create(&m).Error; err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("create", "Voucher")})
}

func (s *Service) findAll(c fiber.Ctx) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	var data []domain.Voucher
	var total int64
	if err := s.db.Model(&domain.Voucher{}).Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return httpx.NotFound("Voucher")
	}
	if err := s.db.Offset((page - 1) * limit).Limit(limit).Find(&data).Error; err != nil {
		return err
	}
	mapped, err := promotionutil.AsViews(data)
	if err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Voucher"), Data: mapped, Paging: httpx.Pagination(page, limit, total)})
}

func (s *Service) byID(c fiber.Ctx) error {
	if err := httpx.UUID(c.Params("id")); err != nil {
		return err
	}
	var m domain.Voucher
	if err := s.db.First(&m, "id = ?", c.Params("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("Voucher")
		}
		return err
	}
	view, err := promotionutil.AsView(m)
	if err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Voucher"), Data: view})
}

func (s *Service) update(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var m domain.Voucher
	if err := s.db.First(&m, "id = ?", id).Error; err != nil {
		return err
	}
	if err := promotionutil.ValidateFields(c, promotionutil.VoucherFields(), true); err != nil {
		return err
	}
	fields, err := promotionutil.BodyFields(c, inputFields...)
	if err != nil {
		return err
	}
	updates := make(map[string]any, len(fields))
	old := map[string]any{
		"code": m.Code, "name": m.Name, "description": promotionutil.NullString(m.Description),
		"type": m.Type, "group": m.Group, "value": promotionutil.NullFloat(m.Value),
		"product_id": promotionutil.NullString(m.ProductID), "quantity": promotionutil.NullInt(m.Quantity),
		"minimum_purchase": m.MinimumPurchase, "is_active": m.IsActive,
		"start_date": m.StartDate, "end_date": m.EndDate,
		"max_redeemed": m.MaxRedeemed, "redeemed_count": m.RedeemedCount,
	}
	for key, raw := range fields {
		switch key {
		case "code", "name":
			value, err := promotionutil.Value[string](raw)
			if err != nil {
				return err
			}
			if key == "code" {
				if err := promotionutil.Text(value, "Code", 1, 50); err != nil {
					return err
				}
			} else if err := promotionutil.Name(value, "Name", 1, 100); err != nil {
				return err
			}
			updates[key] = value
		case "type", "group":
			value, err := promotionutil.Value[string](raw)
			if err != nil {
				return err
			}
			if key == "type" {
				if err := promotionutil.Enum(value, "Discount Type", "PERCENTAGE", "FIXED", "FREE_ITEM"); err != nil {
					return err
				}
			} else if err := promotionutil.Enum(value, "Group", "CHECKOUT", "SHIPPING"); err != nil {
				return err
			}
			updates[key] = value
		case "description":
			value, err := promotionutil.Nullable[string](raw)
			if err != nil {
				return err
			}
			if value != nil {
				if err := promotionutil.Text(*value, "Description", 1, 255); err != nil {
					return err
				}
			}
			updates["description"] = promotionutil.NullString(value)
		case "value":
			value, err := promotionutil.Nullable[float64](raw)
			if err != nil {
				return err
			}
			updates["value"] = promotionutil.NullFloat(value)
		case "productId":
			value, err := promotionutil.NullableUUID(raw, "Product ID")
			if err != nil {
				return err
			}
			// Updating to an actual product requires the same existence lookup
			// as the public API; clearing the optional FK is permitted.
			if value != nil {
				var product domain.Product
				if err := s.db.First(&product, "id = ?", value).Error; err != nil {
					return err
				}
			}
			updates["product_id"] = value
		case "quantity":
			value, err := promotionutil.Nullable[float64](raw)
			if err != nil {
				return err
			}
			updates["quantity"] = promotionutil.NullInt(promotionutil.NullableInteger(value))
		case "minimumPurchase":
			value, err := promotionutil.Value[float64](raw)
			if err != nil {
				return err
			}
			updates["minimum_purchase"] = value
		case "isActive":
			value, err := promotionutil.Value[bool](raw)
			if err != nil {
				return err
			}
			updates["is_active"] = value
		case "maxRedeemed", "redeemedCount":
			value, err := promotionutil.Value[float64](raw)
			if err != nil {
				return err
			}
			if key == "maxRedeemed" {
				updates["max_redeemed"] = promotionutil.Integer(value)
			} else {
				updates["redeemed_count"] = promotionutil.Integer(value)
			}
		case "startDate", "endDate":
			date, err := promotionutil.OptionalDate(raw)
			if err != nil {
				return err
			}
			if key == "startDate" {
				updates["start_date"] = date
			} else {
				updates["end_date"] = date
			}
		}
	}
	if err := promotionutil.ValidateUpdateFields(old, updates, promotionutil.VoucherFields()...); err != nil {
		return err
	}
	if err := s.db.Model(&m).Updates(updates).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "Voucher")})
}

func (s *Service) remove(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	result := s.db.Model(&domain.Voucher{}).Where("id = ?", id).Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return httpx.NotFound("Voucher")
	}
	return c.JSON(httpx.Response{Message: httpx.Message("delete", "Voucher")})
}
