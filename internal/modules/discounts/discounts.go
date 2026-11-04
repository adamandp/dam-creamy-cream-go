package discounts

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/discounts")
	r.Post("/", s.create)
	r.Get("/", s.findAll)
	r.Get("/:id", s.byID)
	r.Patch("/:id", s.update)
	r.Delete("/:id", s.remove)
}

type CreateInput struct {
	Name          string             `json:"name"`
	DiscountType  string             `json:"discountType"`
	Value         *float64           `json:"value"`
	MaxApplied    int                `json:"maxApplied"`
	MaxRedeemed   int                `json:"maxRedeemed"`
	RedeemedCount int                `json:"redeemedCount"`
	Description   *string            `json:"description"`
	ProductID     *string            `json:"productId"`
	Quantity      *int               `json:"quantity"`
	StartDate     promotionutil.Date `json:"startDate"`
	EndDate       promotionutil.Date `json:"endDate"`
}

var inputFields = []string{
	"name", "discountType", "value", "maxApplied", "maxRedeemed",
	"redeemedCount", "description", "productId", "quantity", "startDate", "endDate",
}

func validate(b CreateInput) error {
	if err := promotionutil.Name(b.Name, "Discount Name", 1, 100); err != nil {
		return err
	}
	if err := promotionutil.Enum(b.DiscountType, "Discount Type", "PERCENTAGE", "FIXED", "FREE_ITEM", "PRICE"); err != nil {
		return err
	}
	if b.Description != nil {
		if err := promotionutil.Text(*b.Description, "Description", 1, 255); err != nil {
			return err
		}
	}
	if b.ProductID != nil {
		if err := httpx.UUID(*b.ProductID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) create(c fiber.Ctx) error {
	var body CreateInput
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	if err := promotionutil.Required(c, inputFields...); err != nil {
		return err
	}
	if err := httpx.RejectNull(c, "name", "discountType", "maxApplied", "maxRedeemed", "redeemedCount"); err != nil {
		return err
	}
	if err := validate(body); err != nil {
		return err
	}
	m := domain.Discount{
		Name: body.Name, DiscountType: body.DiscountType, Value: body.Value,
		MaxApplied: body.MaxApplied, MaxRedeemed: body.MaxRedeemed,
		RedeemedCount: body.RedeemedCount, Description: body.Description,
		ProductID: body.ProductID, Quantity: body.Quantity,
		StartDate: body.StartDate.Time, EndDate: body.EndDate.Time,
	}
	if err := s.db.Create(&m).Error; err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("create", "Discount")})
}

func (s *Service) findAll(c fiber.Ctx) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	var data []domain.Discount
	var total int64
	if err := s.db.Model(&domain.Discount{}).Count(&total).Error; err != nil {
		return err
	}
	// Preserve the current public error/message contract.
	if total <= 0 {
		return httpx.NotFound("Users")
	}
	if err := s.db.Offset((page - 1) * limit).Limit(limit).Find(&data).Error; err != nil {
		return err
	}
	mapped, err := promotionutil.AsViews(data)
	if err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "User"), Data: mapped, Paging: httpx.Pagination(page, limit, total)})
}

func (s *Service) byID(c fiber.Ctx) error {
	if err := httpx.UUID(c.Params("id")); err != nil {
		return err
	}
	var m domain.Discount
	if err := s.db.First(&m, "id = ?", c.Params("id")).Error; err != nil {
		return err
	}
	view, err := promotionutil.AsView(m)
	if err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Discount"), Data: view})
}

func (s *Service) update(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var m domain.Discount
	if err := s.db.First(&m, "id = ?", id).Error; err != nil {
		return err
	}
	fields, err := promotionutil.BodyFields(c, inputFields...)
	if err != nil {
		return err
	}
	updates := make(map[string]any, len(fields))
	old := map[string]any{
		"name": m.Name, "discount_type": m.DiscountType, "value": promotionutil.NullFloat(m.Value),
		"max_applied": m.MaxApplied, "max_redeemed": m.MaxRedeemed, "redeemed_count": m.RedeemedCount,
		"description": promotionutil.NullString(m.Description), "product_id": promotionutil.NullString(m.ProductID),
		"quantity": promotionutil.NullInt(m.Quantity), "start_date": m.StartDate, "end_date": m.EndDate,
	}
	for key, raw := range fields {
		switch key {
		case "name":
			value, err := promotionutil.Value[string](raw)
			if err != nil {
				return err
			}
			if err := promotionutil.Name(value, "Discount Name", 1, 100); err != nil {
				return err
			}
			updates["name"] = value
		case "discountType":
			value, err := promotionutil.Value[string](raw)
			if err != nil {
				return err
			}
			if err := promotionutil.Enum(value, "Discount Type", "PERCENTAGE", "FIXED", "FREE_ITEM", "PRICE"); err != nil {
				return err
			}
			updates["discount_type"] = value
		case "value":
			value, err := promotionutil.Nullable[float64](raw)
			if err != nil {
				return err
			}
			updates["value"] = promotionutil.NullFloat(value)
		case "maxApplied", "maxRedeemed", "redeemedCount":
			value, err := promotionutil.Value[int](raw)
			if err != nil {
				return err
			}
			column := map[string]string{"maxApplied": "max_applied", "maxRedeemed": "max_redeemed", "redeemedCount": "redeemed_count"}[key]
			updates[column] = value
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
		case "productId":
			value, err := promotionutil.NullableUUID(raw, "Product ID")
			if err != nil {
				return err
			}
			updates["product_id"] = value
		case "quantity":
			value, err := promotionutil.Nullable[int](raw)
			if err != nil {
				return err
			}
			updates["quantity"] = promotionutil.NullInt(value)
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
	if err := httpx.ValidateUpdate(old, updates); err != nil {
		return err
	}
	if err := s.db.Model(&m).Updates(updates).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "Discount")})
}

func (s *Service) remove(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var m domain.Discount
	if err := s.db.First(&m, "id = ?", id).Error; err != nil {
		return err
	}
	if err := s.db.Delete(&m).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("delete", "Discount")})
}
