package products

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/pricing"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	upload   infrastructure.ImageStore
	sessions *sessions.Service
}

func NewService(db *gorm.DB, upload infrastructure.ImageStore, ses *sessions.Service) *Service {
	return &Service{db: db, upload: upload, sessions: ses}
}
func (s *Service) Routes(app fiber.Router) {
	p := app.Group("/products")
	p.Post("/", s.sessions.RequireAdmin, s.create)
	p.Get("/", s.findAll)
	p.Get("/catalog", s.catalog)
	p.Patch("/:id", s.sessions.RequireAdmin, s.update)
	p.Patch("/:id/image", s.sessions.RequireAdmin, s.updateImage)
	p.Delete("/:id", s.sessions.RequireAdmin, s.remove)
	details := app.Group("/product")
	details.Get("/information/:id", s.information)
	details.Get("/:id", s.detail)
}

type CreateInput struct {
	CategoryID  string  `json:"categoryId"`
	DiscountID  *string `json:"discountId"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Price       float64 `json:"price"`
	IsActive    *bool   `json:"isActive"`
}
type UpdateInput struct {
	CategoryID  *string  `json:"categoryId"`
	DiscountID  *string  `json:"discountId"`
	Name        *string  `json:"name"`
	Description *string  `json:"description"`
	Price       *float64 `json:"price"`
	IsActive    *bool    `json:"isActive"`
}

func checkPrice(v float64) error {
	if v < 1000 || v > 1000000 {
		return fiber.NewError(400, "price must be between 1000 and 1000000")
	}
	return nil
}
func (s *Service) categoryExists(id string) error {
	var count int64
	if err := s.db.Model(&domain.Category{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return httpx.NotFound("category")
	}
	return nil
}
func (s *Service) create(c fiber.Ctx) error {
	var b CreateInput
	if err := httpx.BindStrict(c, &b); err != nil {
		return err
	}
	if err := httpx.Required(c, "categoryId", "name", "description", "price", "isActive"); err != nil {
		return err
	}
	if err := httpx.UUID(b.CategoryID); err != nil {
		return err
	}
	if b.DiscountID != nil {
		if err := httpx.UUID(*b.DiscountID); err != nil {
			return err
		}
	}
	if err := httpx.NonBlank(b.Name, 1, 100, "name"); err != nil {
		return err
	}
	if b.Description != nil {
		if err := httpx.NonBlank(*b.Description, 1, 255, "description"); err != nil {
			return err
		}
	}
	if err := checkPrice(b.Price); err != nil {
		return err
	}
	if b.IsActive == nil {
		return fiber.NewError(400, "isActive is required")
	}
	if err := s.categoryExists(b.CategoryID); err != nil {
		return err
	}
	m := domain.Product{CategoryID: b.CategoryID, DiscountID: b.DiscountID, Name: b.Name, Description: b.Description, Price: b.Price, IsActive: *b.IsActive}
	if err := s.db.Create(&m).Error; err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("create", "Product")})
}
func (s *Service) findAll(c fiber.Ctx) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	var data []domain.Product
	var total int64
	if err := s.db.Model(&domain.Product{}).Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return httpx.NotFound("Product")
	}
	if err := s.db.Offset((page - 1) * limit).Limit(limit).Find(&data).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Product"), Data: data, Paging: httpx.Pagination(page, limit, total)})
}
func (s *Service) update(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var m domain.Product
	if err := s.db.First(&m, "id = ?", id).Error; err != nil {
		return err
	}
	var b UpdateInput
	if err := httpx.BindStrict(c, &b); err != nil {
		return err
	}
	if err := httpx.RejectNull(c, "categoryId", "name", "price", "isActive"); err != nil {
		return err
	}
	updates := map[string]any{}
	if b.CategoryID != nil {
		if err := httpx.UUID(*b.CategoryID); err != nil {
			return err
		}
		if err := s.categoryExists(*b.CategoryID); err != nil {
			return err
		}
		updates["category_id"] = *b.CategoryID
	}
	if b.DiscountID != nil {
		if err := httpx.UUID(*b.DiscountID); err != nil {
			return err
		}
		updates["discount_id"] = *b.DiscountID
	}
	if b.Name != nil {
		if err := httpx.NonBlank(*b.Name, 1, 100, "name"); err != nil {
			return err
		}
		updates["name"] = *b.Name
	}
	if b.Description != nil {
		if err := httpx.NonBlank(*b.Description, 1, 255, "description"); err != nil {
			return err
		}
		updates["description"] = *b.Description
	}
	if b.Price != nil {
		if err := checkPrice(*b.Price); err != nil {
			return err
		}
		updates["price"] = *b.Price
	}
	if b.IsActive != nil {
		updates["is_active"] = *b.IsActive
	}
	if httpx.ExplicitNull(c, "description") {
		updates["description"] = nil
	}
	if httpx.ExplicitNull(c, "discountId") {
		updates["discount_id"] = nil
	}
	if err := httpx.ValidateUpdate(map[string]any{"category_id": m.CategoryID, "discount_id": valueOrNil(m.DiscountID), "name": m.Name, "description": valueOrNil(m.Description), "price": m.Price, "is_active": m.IsActive}, updates); err != nil {
		return err
	}
	if len(updates) > 0 {
		if err := s.db.Model(&m).Updates(updates).Error; err != nil {
			return err
		}
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "Product")})
}
func (s *Service) remove(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	result := s.db.Delete(&domain.Product{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return httpx.NotFound("Product")
	}
	return c.JSON(httpx.Response{Message: httpx.Message("delete", "Product")})
}
func validatedImage(f *multipart.FileHeader) ([]byte, error) {
	if f == nil {
		return nil, fiber.NewError(400, "file is required")
	}
	if f.Size > 2*1024*1024 {
		return nil, fiber.NewError(400, "file too large")
	}
	mimetype := f.Header.Get("Content-Type")
	switch mimetype {
	case "image/jpg", "image/jpeg", "image/png", "image/webp":
	default:
		return nil, fiber.NewError(400, "Only image files are allowed!")
	}
	src, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()
	bytes, err := io.ReadAll(io.LimitReader(src, 2*1024*1024+1))
	if err != nil {
		return nil, err
	}
	if len(bytes) > 2*1024*1024 {
		return nil, fiber.NewError(400, "file too large")
	}
	return bytes, nil
}
func (s *Service) updateImage(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	f, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(400, "file is required")
	}
	content, err := validatedImage(f)
	if err != nil {
		return err
	}
	var m domain.Product
	if err := s.db.First(&m, "id = ?", id).Error; err != nil {
		return err
	}
	previousImageURL := ""
	if m.ImageURL != nil {
		previousImageURL = *m.ImageURL
	}
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()
	imageURL, err := s.upload.Upload(ctx, "products", id, content)
	if err != nil {
		if errors.Is(err, infrastructure.ErrInvalidImage) || errors.Is(err, infrastructure.ErrImageTooLarge) {
			return fiber.NewError(400, "Only image files are allowed!")
		}
		return fiber.NewError(400, "📤 Oops! Failed to upload "+id+". Please try again later! 🚧")
	}
	if err := s.db.WithContext(ctx).Model(&m).Update("image_url", imageURL).Error; err != nil {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if cleanupErr := s.upload.Delete(cleanupCtx, imageURL); cleanupErr != nil {
			slog.Warn("failed to remove orphaned product image", "error", cleanupErr)
		}
		return err
	}
	if previousImageURL != "" {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := s.upload.Delete(cleanupCtx, previousImageURL); err != nil {
			slog.Warn("failed to remove previous product image", "error", err)
		}
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "Product Image")})
}

type CatalogItem struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	ImageURL      string   `json:"imageUrl"`
	Description   string   `json:"description"`
	Rate          float64  `json:"rate"`
	Price         float64  `json:"price"`
	DiscountPrice *float64 `json:"discountPrice"`
	DiscountType  *string  `json:"discountType"`
	DiscountValue *float64 `json:"discountValue"`
}
type DetailItem struct {
	CatalogItem
	Category string `json:"category"`
	Sales    int64  `json:"sales"`
}
type ReviewRating struct {
	ProductID string  `gorm:"column:product_id"`
	Average   float64 `gorm:"column:average"`
}
type ActiveDiscount struct {
	ProductID string   `gorm:"column:product_id"`
	Type      string   `gorm:"column:discount_type"`
	Value     *float64 `gorm:"column:value"`
}

func (s *Service) ratings(ids []string) (map[string]float64, error) {
	result := map[string]float64{}
	if len(ids) == 0 {
		return result, nil
	}
	var rows []ReviewRating
	err := s.db.Model(&domain.Review{}).Select("product_id, AVG(rating) AS average").Where("product_id IN ?", ids).Group("product_id").Scan(&rows).Error
	for _, r := range rows {
		result[r.ProductID] = r.Average
	}
	return result, err
}
func (s *Service) discounts(ids []string) (map[string]ActiveDiscount, error) {
	result := map[string]ActiveDiscount{}
	if len(ids) == 0 {
		return result, nil
	}
	var rows []ActiveDiscount
	now := time.Now().UTC()
	err := s.db.Table("product_discounts pd").Select("pd.product_id, d.discount_type, d.value").
		Joins("JOIN discounts d ON d.id = pd.discount_id").
		Where("pd.product_id IN ? AND "+pricing.ActiveAssociation, ids, now, now).
		Scan(&rows).Error
	for _, r := range rows {
		result[r.ProductID] = r
	}
	return result, err
}
func summary(p domain.Product, rate float64, d ActiveDiscount) CatalogItem {
	item := CatalogItem{ID: p.ID.ID, Name: p.Name, Rate: rate, Price: p.Price}
	if p.ImageURL != nil {
		item.ImageURL = *p.ImageURL
	}
	if p.Description != nil {
		item.Description = *p.Description
	}
	if discounted := pricing.DiscountPrice(p.Price, d.Type, d.Value); discounted != nil {
		item.DiscountType = &d.Type
		item.DiscountValue = d.Value
		item.DiscountPrice = discounted
	}
	return item
}
func (s *Service) catalog(c fiber.Ctx) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	// ProductQuerySchema is strict in NestJS. Track presence separately from
	// value: z.coerce.number("") is 0, whereas an absent bound is omitted.
	allowed := map[string]bool{"page": true, "limit": true, "q": true, "sort": true, "cat": true, "min": true, "max": true}
	present := map[string]bool{}
	var unexpected []string
	c.Request().URI().QueryArgs().VisitAll(func(k, _ []byte) {
		key := string(k)
		present[key] = true
		if !allowed[key] {
			unexpected = append(unexpected, key)
		}
	})
	if len(unexpected) > 0 {
		return fiber.NewError(400, "Unrecognized catalog query parameter")
	}
	q := c.Query("q")
	sort := c.Query("sort")
	cat := c.Query("cat")
	if present["q"] && (httpx.JSStringLength(q) < 1 || httpx.JSStringLength(q) > 100) {
		return fiber.NewError(400, "q must be 1-100 characters")
	}
	if present["sort"] && sort != "asc" && sort != "desc" && sort != "default" {
		return fiber.NewError(400, "sort must be asc, desc or default")
	}
	if present["cat"] && (httpx.JSStringLength(cat) < 1 || httpx.JSStringLength(cat) > 100) {
		return fiber.NewError(400, "cat must be 1-100 characters")
	}
	var categoryIDs []string
	if cat != "" {
		categoryIDs = strings.Split(cat, ",")
		for _, id := range categoryIDs {
			if err := httpx.UUID(id); err != nil {
				return err
			}
		}
	}
	type priceBound struct {
		cmp   string
		value float64
	}
	var bounds []priceBound
	for _, cond := range []struct{ key, cmp string }{{"min", ">="}, {"max", "<="}} {
		if !present[cond.key] {
			continue
		}
		raw := c.Query(cond.key)
		n, err := strconv.ParseFloat(raw, 64)
		if raw == "" {
			n, err = 0, nil
		}
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n > 1000000 {
			return fiber.NewError(400, fmt.Sprintf("%s must be within 0..1000000", cond.key))
		}
		bounds = append(bounds, priceBound{cmp: cond.cmp, value: n})
	}
	db := s.db.Model(&domain.Product{}).Where("is_active = ?", true)
	if q != "" {
		// Prisma's PostgreSQL `contains` compiles to ILIKE with the same
		// LIKE metacharacter behavior. Preserve it for 1:1 source parity.
		needle := "%" + q + "%"
		db = db.Where("(name ILIKE ? OR description ILIKE ?)", needle, needle)
	}
	if len(categoryIDs) > 0 {
		db = db.Where("category_id IN ?", categoryIDs)
	}
	for _, bound := range bounds {
		db = db.Where("price "+bound.cmp+" ?", bound.value)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return err
	}
	if total == 0 && page == 1 {
		return httpx.NotFound("Product")
	}
	sortExpr := "created_at desc"
	if sort == "asc" {
		sortExpr = "price asc"
	} else if sort == "desc" {
		sortExpr = "price desc"
	}
	var products []domain.Product
	if err := db.Order(sortExpr).Offset((page - 1) * limit).Limit(limit).Find(&products).Error; err != nil {
		return err
	}
	ids := make([]string, 0, len(products))
	for _, p := range products {
		ids = append(ids, p.ID.ID)
	}
	ratings, err := s.ratings(ids)
	if err != nil {
		return err
	}
	discounts, err := s.discounts(ids)
	if err != nil {
		return err
	}
	data := make([]CatalogItem, 0, len(products))
	for _, p := range products {
		data = append(data, summary(p, ratings[p.ID.ID], discounts[p.ID.ID]))
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Product"), Data: data, Paging: httpx.Pagination(page, limit, total)})
}
func (s *Service) detail(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var m domain.Product
	if err := s.db.First(&m, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("Product")
		}
		return err
	}
	ratings, err := s.ratings([]string{id})
	if err != nil {
		return err
	}
	discounts, err := s.discounts([]string{id})
	if err != nil {
		return err
	}
	var category domain.Category
	if err := s.db.First(&category, "id = ?", m.CategoryID).Error; err != nil {
		return err
	}
	var sales int64
	if err := s.db.Model(&domain.OrderItem{}).Select("COALESCE(SUM(quantity),0)").Where("product_id = ?", id).Scan(&sales).Error; err != nil {
		return err
	}
	item := DetailItem{CatalogItem: summary(m, ratings[id], discounts[id]), Category: category.Name, Sales: sales}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Product"), Data: item})
}
func (s *Service) information(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var info domain.ProductInformation
	if err := s.db.Where("product_id = ?", id).First(&info).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("Product Information")
		}
		return err
	}
	var m domain.Product
	if err := s.db.First(&m, "id = ?", id).Error; err != nil {
		return err
	}
	var highlights []string
	var serving []string
	var details struct {
		Size      string `json:"size"`
		Storage   string `json:"storage"`
		ShelfLife string `json:"shelfLife"`
	}
	if err := json.Unmarshal(info.Highlights, &highlights); err != nil {
		return err
	}
	if err := json.Unmarshal(info.ServingSuggestions, &serving); err != nil {
		return err
	}
	if err := json.Unmarshal(info.ProductDetails, &details); err != nil {
		return err
	}
	description := ""
	if m.Description != nil {
		description = *m.Description
	}
	lines := []string{description, "", "🍦 Highlights"}
	for _, v := range highlights {
		lines = append(lines, "• "+v)
	}
	lines = append(lines, "", "🍨 Serving Suggestions")
	for _, v := range serving {
		lines = append(lines, "• "+v)
	}
	lines = append(lines, "", "📦 Product Details", "• Size: "+details.Size, "• Storage: "+details.Storage, "• Shelf life: "+details.ShelfLife, "", "⚡ Why Choose This Product?", info.WhyChoose)
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Product"), Data: fiber.Map{"information": strings.TrimSpace(strings.Join(lines, "\n"))}})
}

func valueOrNil(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}
