package recommendations

import (
	"sort"
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func (s *Service) decorate(c fiber.Ctx, products []domain.Product) ([]recommendation, error) {
	result := make([]recommendation, len(products))
	if len(products) == 0 {
		return result, nil
	}
	ids := make([]string, 0, len(products))
	byID := make(map[string]int, len(products))
	for i, product := range products {
		result[i].Product = product
		ids = append(ids, product.ID.ID)
		byID[product.ID.ID] = i
	}
	var reviews []struct {
		ProductID string
		Rate      float64
	}
	err := s.db.WithContext(c.Context()).Table("reviews").
		Select("product_id, AVG(rating)::double precision AS rate").Where("product_id IN ?", ids).
		Group("product_id").Scan(&reviews).Error
	if err != nil {
		return nil, err
	}
	for _, r := range reviews {
		result[byID[r.ProductID]].Rate = r.Rate
	}
	var sales []struct {
		ProductID string
		Total     int64
	}
	err = s.db.WithContext(c.Context()).Table("order_items").
		Select("product_id, COUNT(*) AS total").Where("product_id IN ?", ids).
		Group("product_id").Scan(&sales).Error
	if err != nil {
		return nil, err
	}
	for _, r := range sales {
		result[byID[r.ProductID]].Sales = r.Total
	}
	var discounts []struct {
		ProductID    string
		DiscountType string
		Value        *float64
	}
	err = s.db.WithContext(c.Context()).Table("product_discounts AS pd").
		Select("pd.product_id, d.discount_type, d.value").
		Joins("JOIN discounts AS d ON d.id = pd.discount_id").
		Where("pd.is_active = ? AND pd.product_id IN ?", true, ids).Scan(&discounts).Error
	if err != nil {
		return nil, err
	}
	for _, d := range discounts {
		result[byID[d.ProductID]].Discount = &discount{Type: d.DiscountType, Value: d.Value}
	}
	return result, nil
}

func (s *Service) classic(c fiber.Ctx) error {
	var products []domain.Product
	err := s.db.WithContext(c.Context()).Model(&domain.Product{}).
		Joins("JOIN category ON category.id = products.category_id").
		Where("products.is_active = ? AND category.name ILIKE ?", true, "%Classic%").
		Find(&products).Error
	if err != nil {
		return err
	}
	if len(products) == 0 {
		return httpx.NotFound("Classic Recommendations")
	}
	rows, err := s.decorate(c, products)
	if err != nil {
		return err
	}
	rankByRatingAndSales(rows)
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Classic Recommendations"), Data: productViews(rows)})
}

func (s *Service) categories(c fiber.Ctx) error {
	var categories []domain.Category
	if err := s.db.WithContext(c.Context()).Where("is_active = ?", true).Find(&categories).Error; err != nil {
		return err
	}
	if len(categories) == 0 {
		return httpx.NotFound("Categories Recommendations")
	}
	data := make([]fiber.Map, 0, len(categories))
	for _, category := range categories {
		// In the source each category includes only the most sold active product.
		var product domain.Product
		result := s.db.WithContext(c.Context()).Model(&domain.Product{}).
			Select("products.*").Joins("LEFT JOIN order_items oi ON oi.product_id = products.id").
			Where("products.category_id = ? AND products.is_active = ?", category.ID.ID, true).
			Group("products.id").Order("COUNT(oi.id) DESC").Limit(1).Find(&product)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			continue
		}
		image := ""
		if product.ImageURL != nil {
			image = *product.ImageURL
		}
		data = append(data, fiber.Map{"id": product.ID.ID, "imageUrl": image, "category": category.Name, "categoryId": category.ID.ID})
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Categories Recommendations"), Data: data})
}

func (s *Service) offers(c fiber.Ctx) error {
	var products []domain.Product
	err := s.db.WithContext(c.Context()).Model(&domain.Product{}).
		Joins("JOIN product_discounts pd ON pd.product_id = products.id").
		Where("products.is_active = ? AND pd.is_active = ?", true, true).Limit(5).Find(&products).Error
	if err != nil {
		return err
	}
	if len(products) == 0 {
		return httpx.NotFound("Offer Recommendations")
	}
	rows, err := s.decorate(c, products)
	if err != nil {
		return err
	}
	// Preserve the original offers comparator: percentage sorting divides
	// discount value by 100 (not by product price), before rating and sales.
	value := func(p recommendation) float64 {
		if p.Discount == nil || p.Discount.Value == nil {
			return 0
		}
		switch p.Discount.Type {
		case "FIXED":
			return *p.Discount.Value
		case "PERCENTAGE":
			return *p.Discount.Value / 100
		default:
			return 0
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if value(rows[i]) != value(rows[j]) {
			return value(rows[i]) > value(rows[j])
		}
		if rows[i].Rate != rows[j].Rate {
			return rows[i].Rate > rows[j].Rate
		}
		return rows[i].Sales > rows[j].Sales
	})
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Offer Recommendations"), Data: productViews(rows)})
}

func (s *Service) related(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var product domain.Product
	if err := s.db.WithContext(c.Context()).First(&product, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("Related Recommendations")
		}
		return err
	}
	var products []domain.Product
	err := s.db.WithContext(c.Context()).Where("is_active = ? AND id <> ? AND category_id = ?", true, id, product.CategoryID).Find(&products).Error
	if err != nil {
		return err
	}
	rows, err := s.decorate(c, products)
	if err != nil {
		return err
	}
	rankByRatingAndSales(rows)
	if len(rows) < 5 {
		exclude := []string{id}
		for _, p := range rows {
			exclude = append(exclude, p.Product.ID.ID)
		}
		var additional []domain.Product
		if err := s.db.WithContext(c.Context()).Where("is_active = ? AND id NOT IN ?", true, exclude).Find(&additional).Error; err != nil {
			return err
		}
		more, err := s.decorate(c, additional)
		if err != nil {
			return err
		}
		rows = append(rows, more...)
		rankByRatingAndSales(rows)
	}
	if len(rows) == 0 {
		return httpx.NotFound("Related Recommendations")
	}
	if len(rows) > 5 {
		rows = rows[:5]
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Related Recommendations"), Data: productViews(rows)})
}
