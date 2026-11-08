package productdiscounts

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
	r := app.Group("/product-discounts")
	r.Post("/", s.sessions.RequireAdmin, s.create)
	r.Get("/", s.findAll)
	r.Get("/product/:productId", s.byProductID)
	r.Get("/:id", s.byID)
	r.Delete("/:id", s.sessions.RequireAdmin, s.remove)
}

type CreateInput struct {
	ProductID  string `json:"productId"`
	DiscountID string `json:"discountId"`
}

func (s *Service) create(c fiber.Ctx) error {
	if err := promotionutil.ValidateFields(c, promotionutil.ProductDiscountFields(), false); err != nil {
		return err
	}
	var body CreateInput
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	if err := promotionutil.Required(c, "productId", "discountId"); err != nil {
		return err
	}
	// Resolve both foreign keys before creating the association, in the
	// same order as the existing public API.
	var product domain.Product
	if err := s.db.First(&product, "id = ?", body.ProductID).Error; err != nil {
		return err
	}
	var discount domain.Discount
	if err := s.db.First(&discount, "id = ?", body.DiscountID).Error; err != nil {
		return err
	}
	m := domain.ProductDiscount{ProductID: body.ProductID, DiscountID: body.DiscountID, IsActive: true}
	if err := s.db.Create(&m).Error; err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("create", "Product Discount")})
}

func (s *Service) findAll(c fiber.Ctx) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	var data []domain.ProductDiscount
	// Count the actual product discounts, fixing legacy pagination.
	var total int64
	if err := s.db.Model(&domain.ProductDiscount{}).Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return httpx.NotFound("Product Discount")
	}
	if err := s.db.Offset((page - 1) * limit).Limit(limit).Find(&data).Error; err != nil {
		return err
	}
	mapped, err := promotionutil.AsViews(data)
	if err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Product Discount"), Data: mapped, Paging: httpx.Pagination(page, limit, total)})
}

func (s *Service) byID(c fiber.Ctx) error {
	if err := httpx.UUID(c.Params("id")); err != nil {
		return err
	}
	var m domain.ProductDiscount
	if err := s.db.First(&m, "id = ?", c.Params("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("Product Discount")
		}
		return err
	}
	view, err := promotionutil.AsView(m)
	if err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Product Discount"), Data: view})
}

func (s *Service) byProductID(c fiber.Ctx) error {
	if err := httpx.UUID(c.Params("productId")); err != nil {
		return err
	}
	var data []domain.ProductDiscount
	if err := s.db.Where("product_id = ?", c.Params("productId")).Find(&data).Error; err != nil {
		return err
	}
	mapped, err := promotionutil.AsViews(data)
	if err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Product Discount"), Data: mapped})
}

func (s *Service) remove(c fiber.Ctx) error {
	if err := httpx.UUID(c.Params("id")); err != nil {
		return err
	}
	result := s.db.Model(&domain.ProductDiscount{}).
		Where("id = ?", c.Params("id")).Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return httpx.NotFound("Product Discount")
	}
	return c.JSON(httpx.Response{Message: httpx.Message("delete", "Product Discount")})
}
