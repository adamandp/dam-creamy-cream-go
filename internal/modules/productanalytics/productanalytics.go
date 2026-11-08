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
	r.Post("/", s.sessions.RequireActor, s.create)
	r.Get("/product/:id/count", s.productCount)
	r.Get("/product/:id", s.byProduct)
	r.Get("/user", s.sessions.RequireActor, s.byUser)
}

var createFields = []promotionutil.Field{
	{Key: "userId", Label: "User id", Kind: "uuid"},
	{Key: "productId", Label: "Product id", Kind: "uuid"},
}

func (s *Service) create(c fiber.Ctx) error {
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
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
	// PostgreSQL stores both IDs as non-null UUIDs. The NestJS "unkown"
	// fallback causes an invalid-UUID database error; reject it at the edge.
	if body.UserID == nil || body.ProductID == nil {
		return fiber.NewError(400, "userId and productId must be valid UUIDs")
	}
	if !actor.Admin && *body.UserID != actor.UserID {
		return httpx.Forbidden()
	}
	m := domain.ProductAnalytics{UserID: *body.UserID, ProductID: *body.ProductID}
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
	// Count exactly the same filtered rows as the page, not the global table.
	var total int64
	if err := s.db.WithContext(c.Context()).Model(&domain.ProductAnalytics{}).Where(predicate, id).Count(&total).Error; err != nil {
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
