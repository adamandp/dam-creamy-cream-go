package carts

import (
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/pricing"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	db      *gorm.DB
	session *sessions.Service
}

func NewService(db *gorm.DB, s *sessions.Service) *Service { return &Service{db: db, session: s} }
func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/carts")
	r.Post("/user/add", s.add)
	r.Post("/user/remove", s.remove)
	r.Get("/user", s.byUser)
}

type Input struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

func validateInput(in Input) error {
	if err := httpx.UUID(in.ProductID); err != nil {
		return err
	}
	if in.Quantity < 1 || in.Quantity > 100 {
		return fiber.NewError(400, "quantity must be between 1 and 100")
	}
	return nil
}
func (s *Service) validateRelated(tx *gorm.DB, userID, productID string) error {
	var count int64
	if err := tx.Model(&domain.User{}).Where("id = ?", userID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return httpx.NotFound("User")
	}
	if err := tx.Model(&domain.Product{}).Where("id = ?", productID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return httpx.NotFound("Product")
	}
	return nil
}
func (s *Service) add(c fiber.Ctx) error {
	claims, err := s.session.CookieUser(c)
	if err != nil {
		return err
	}
	var body Input
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	if err := validateInput(body); err != nil {
		return err
	}
	err = s.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		if err := s.validateRelated(tx, claims.Subject, body.ProductID); err != nil {
			return err
		}
		var cart domain.Cart
		if err := tx.Where("user_id = ?", claims.Subject).First(&cart).Error; err != nil {
			return err
		}
		item := domain.CartItem{CartID: cart.ID.ID, ProductID: body.ProductID, Quantity: body.Quantity}
		return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "cart_id"}, {Name: "product_id"}}, DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("cart_items.quantity + EXCLUDED.quantity"), "updated_at": time.Now().UTC()})}).Create(&item).Error
	})
	if err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("create", "Cart")})
}
func (s *Service) remove(c fiber.Ctx) error {
	claims, err := s.session.CookieUser(c)
	if err != nil {
		return err
	}
	var body Input
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	if err := validateInput(body); err != nil {
		return err
	}
	err = s.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		if err := s.validateRelated(tx, claims.Subject, body.ProductID); err != nil {
			return err
		}
		var cart domain.Cart
		if err := tx.Where("user_id = ?", claims.Subject).First(&cart).Error; err != nil {
			return err
		}
		var item domain.CartItem
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("cart_id = ? AND product_id = ?", cart.ID.ID, body.ProductID).First(&item).Error; err != nil {
			return err
		}
		if item.Quantity <= body.Quantity {
			return tx.Delete(&item).Error
		}
		return tx.Model(&item).Update("quantity", item.Quantity-body.Quantity).Error
	})
	if err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("delete", "Cart")})
}

type CartRow struct {
	ProductID     string     `gorm:"column:product_id"`
	Quantity      int        `gorm:"column:quantity"`
	Name          string     `gorm:"column:name"`
	ImageURL      *string    `gorm:"column:image_url"`
	Category      string     `gorm:"column:category"`
	Price         float64    `gorm:"column:price"`
	DiscountType  *string    `gorm:"column:discount_type"`
	DiscountValue *float64   `gorm:"column:discount_value"`
	StartDate     *time.Time `gorm:"column:start_date"`
	EndDate       *time.Time `gorm:"column:end_date"`
}
type CartResponse struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	ImageURL      *string  `json:"imageUrl"`
	Category      string   `json:"category"`
	Price         float64  `json:"price"`
	DiscountPrice *float64 `json:"discountPrice"`
	Qty           int      `json:"qty"`
}

func (s *Service) byUser(c fiber.Ctx) error {
	claims, err := s.session.CookieUser(c)
	if err != nil {
		return err
	}
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	var total int64
	err = s.db.Table("cart_items ci").Joins("JOIN carts ca ON ca.id = ci.cart_id").Where("ca.user_id = ?", claims.Subject).Count(&total).Error
	if err != nil {
		return err
	}
	data := make([]CartResponse, 0)
	if total == 0 {
		return c.JSON(httpx.Response{Message: httpx.Message("get", "Cart"), Data: data, Paging: httpx.Pagination(page, limit, total)})
	}
	var rows []CartRow
	now := time.Now().UTC()
	err = s.db.Table("cart_items ci").Select(`ci.product_id, ci.quantity, p.name, p.image_url, cat.name AS category, p.price, d.discount_type, d.value AS discount_value`).
		Joins("JOIN carts ca ON ca.id = ci.cart_id").
		Joins("JOIN products p ON p.id = ci.product_id").
		Joins("JOIN category cat ON cat.id = p.category_id").
		Joins("LEFT JOIN product_discounts pd ON pd.product_id = p.id").
		Joins("LEFT JOIN discounts d ON d.id = pd.discount_id AND "+pricing.ActiveAssociation, now, now).
		Where("ca.user_id = ?", claims.Subject).Offset((page - 1) * limit).Limit(limit).Scan(&rows).Error
	if err != nil {
		return err
	}
	for _, r := range rows {
		item := CartResponse{ID: r.ProductID, Name: r.Name, ImageURL: r.ImageURL, Category: r.Category, Price: r.Price, Qty: r.Quantity}
		if r.DiscountType != nil {
			item.DiscountPrice = pricing.DiscountPrice(r.Price, *r.DiscountType, r.DiscountValue)
		}
		data = append(data, item)
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "User Cart"), Data: data, Paging: httpx.Pagination(page, limit, total)})
}
