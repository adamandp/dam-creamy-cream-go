package productanalytics

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/legacyviews"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	sessions *sessions.Service
}

func NewService(db *gorm.DB, ses *sessions.Service) *Service {
	return &Service{db: db, sessions: ses}
}

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/product-analytics")
	r.Post("/", s.create)
	r.Get("/product/:id/count", s.productCount)
	r.Get("/product/:id", s.byProduct)
	r.Get("/user", s.sessions.RequireActor, s.byUser)
}

var createFields = []promotionutil.Field{
	{Key: "userId", Label: "User id", Kind: "uuid", Nullable: true},
	{Key: "productId", Label: "Product id", Kind: "uuid", Nullable: true},
}

func (s *Service) create(c fiber.Ctx) error {
	if err := promotionutil.ValidateFields(c, createFields, false); err != nil {
		return err
	}
	var body struct {
		UserID    *string `json:"userId"`
		ProductID *string `json:"productId"`
	}
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	userID, productID := "unkown", "unkown" // Original NestJS spelling and null fallback.
	if body.UserID != nil {
		userID = *body.UserID
	}
	if body.ProductID != nil {
		productID = *body.ProductID
	}
	m := domain.ProductAnalytics{UserID: userID, ProductID: productID}
	if err := s.db.WithContext(c.Context()).Create(&m).Error; err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("create", "Product Analytic")})
}

func (s *Service) byProduct(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	return s.list(c, "product_id = ?", id)
}

func (s *Service) byUser(c fiber.Ctx) error {
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	return s.list(c, "user_id = ?", actor.UserID)
}

func (s *Service) list(c fiber.Ctx, predicate string, id string) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	var rows []domain.ProductAnalytics
	if err := s.db.WithContext(c.Context()).Where(predicate, id).Offset((page - 1) * limit).Limit(limit).Find(&rows).Error; err != nil {
		return err
	}
	// The legacy service deliberately reports the global table count even for
	// per-product and per-user pages. Keep this for parity, not as a correction.
	var total int64
	if err := s.db.WithContext(c.Context()).Model(&domain.ProductAnalytics{}).Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return httpx.NotFound("Product Analytic")
	}
	data := make([]fiber.Map, 0, len(rows))
	for _, row := range rows {
		data = append(data, legacyviews.Analytics(row))
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Product Analytic"), Data: data, Paging: httpx.Pagination(page, limit, total)})
}

func (s *Service) productCount(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var total int64
	if err := s.db.WithContext(c.Context()).Model(&domain.ProductAnalytics{}).Where("product_id = ?", id).Count(&total).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Product Analytic"), Data: fiber.Map{"count": total}})
}
