package carts

import (
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/pricing"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

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
		// The unique upsert is atomic. Its WHERE condition protects Prisma's
		// PostgreSQL INTEGER column from overflowing after repeated additions.
		// Checking the old quantity in application code would race another add.
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "cart_id"}, {Name: "product_id"}},
			DoUpdates: clause.Assignments(map[string]any{"quantity": gorm.Expr("cart_items.quantity + EXCLUDED.quantity"), "updated_at": time.Now().UTC()}),
			Where:     clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "cart_items.quantity <= 2147483647 - EXCLUDED.quantity"}}},
		}).Create(&item)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return fiber.NewError(409, "Cart quantity exceeds inventory limits")
		}
		return nil
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
