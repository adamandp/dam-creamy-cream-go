package reviews

import (
	"encoding/json"
	"time"

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

var createFields = []promotionutil.Field{
	{Key: "userId", Label: "Role ID", Kind: "uuid"},
	{Key: "productId", Label: "Product ID", Kind: "uuid"},
	{Key: "rating", Label: "Rating", Kind: "number"},
	{Key: "comment", Label: "Review", Kind: "string", Min: 1, Max: 255, Nullable: true},
}

var updateFields = []promotionutil.Field{
	{Key: "comment", Label: "Review", Kind: "string", Min: 1, Max: 255, Nullable: true},
	{Key: "rating", Label: "Rating", Kind: "number"},
}

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/reviews")
	r.Post("/", s.sessions.RequireActor, s.create)
	r.Get("/", s.findAll)
	r.Get("/product/:id", s.byProduct)
	r.Get("/me", s.sessions.RequireActor, s.byUser)
	r.Get("/:id", s.byID)
	r.Patch("/:id", s.sessions.RequireActor, s.update)
	r.Delete("/:id", s.sessions.RequireActor, s.remove)
}

func checkRating(rating float64) error {
	if rating >= 1 && rating <= 5 {
		return nil
	}
	tooSmall := rating < 1
	code, bound, message := "too_big", 5, "🚀 Whoa! Rating is too long. Maximum allowed is 5 characters! ✍️ {Maximum Length}"
	if tooSmall {
		code, bound, message = "too_small", 1, "🔡 Rating must be at least 1 characters. Keep going! 💪 {Minimum Length}"
	}
	issue := map[string]any{"code": code, "origin": "number", "inclusive": true, "message": message, "path": []string{"rating"}}
	if tooSmall {
		issue["minimum"] = bound
	} else {
		issue["maximum"] = bound
	}
	return httpx.ZodIssue(issue)
}

func (s *Service) create(c fiber.Ctx) error {
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	if err := promotionutil.ValidateFields(c, createFields, false); err != nil {
		return err
	}
	var in struct {
		UserID    string  `json:"userId"`
		ProductID string  `json:"productId"`
		Rating    float64 `json:"rating"`
		Comment   *string `json:"comment"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if !actor.Admin && in.UserID != actor.UserID {
		return httpx.Forbidden()
	}
	if err := checkRating(in.Rating); err != nil {
		return err
	}
	var user domain.User
	if err := s.db.WithContext(c.Context()).First(&user, "id = ?", in.UserID).Error; err != nil {
		return err
	}
	var product domain.Product
	if err := s.db.WithContext(c.Context()).First(&product, "id = ?", in.ProductID).Error; err != nil {
		return err
	}
	// Prisma's Int field rejects a fractional rating at persistence time.
	if in.Rating != float64(int(in.Rating)) {
		return fiber.NewError(500, "Internal server error")
	}
	m := domain.Review{UserID: in.UserID, ProductID: in.ProductID, Rating: int(in.Rating), Comment: in.Comment}
	if err := s.db.WithContext(c.Context()).Create(&m).Error; err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("create", "Review")})
}

func (s *Service) findAll(c fiber.Ctx) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	var total int64
	if err := s.db.WithContext(c.Context()).Model(&domain.Review{}).Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return httpx.NotFound("Review")
	}
	var rows []domain.Review
	if err := s.db.WithContext(c.Context()).Offset((page - 1) * limit).Limit(limit).Find(&rows).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Review"), Data: mapReviews(rows), Paging: httpx.Pagination(page, limit, total)})
}

func mapReviews(rows []domain.Review) []fiber.Map {
	data := make([]fiber.Map, 0, len(rows))
	for _, row := range rows {
		data = append(data, legacyviews.Review(row))
	}
	return data
}

func (s *Service) byID(c fiber.Ctx) error {
	var row domain.Review
	if err := s.db.WithContext(c.Context()).First(&row, "id = ?", c.Params("id")).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("Review")
		}
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Review"), Data: legacyviews.Review(row)})
}

func (s *Service) byUser(c fiber.Ctx) error {
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	var rows []domain.Review
	if err := s.db.WithContext(c.Context()).Where("user_id = ?", actor.UserID).Offset((page - 1) * limit).Limit(limit).Find(&rows).Error; err != nil {
		return err
	}
	// Pagination totals must describe the same user-scoped result set.
	var total int64
	if err := s.db.WithContext(c.Context()).Model(&domain.Review{}).Where("user_id = ?", actor.UserID).Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return httpx.NotFound("Review")
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Review"), Data: mapReviews(rows), Paging: httpx.Pagination(page, limit, total)})
}

func (s *Service) byProduct(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	var product domain.Product
	if err := s.db.WithContext(c.Context()).First(&product, "id = ?", id).Error; err != nil {
		return err
	}
	var rows []struct {
		ID        string    `gorm:"column:id"`
		Rating    int       `gorm:"column:rating"`
		Comment   *string   `gorm:"column:comment"`
		CreatedAt time.Time `gorm:"column:created_at"`
		Username  string    `gorm:"column:username"`
		ImageURL  *string   `gorm:"column:image_url"`
	}
	err = s.db.WithContext(c.Context()).Table("reviews").
		Select("reviews.id, reviews.rating, reviews.comment, reviews.created_at, users.username, users.image_url").
		Joins("JOIN users ON users.id = reviews.user_id").Where("reviews.product_id = ?", id).
		Offset((page - 1) * limit).Limit(limit).Scan(&rows).Error
	if err != nil {
		return err
	}
	var total int64
	if err := s.db.WithContext(c.Context()).Model(&domain.Review{}).Where("product_id = ?", id).Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return httpx.NotFound("Review")
	}
	data := make([]fiber.Map, 0, len(rows))
	for _, r := range rows {
		data = append(data, fiber.Map{"id": r.ID, "rate": r.Rating, "comment": r.Comment, "createdAt": promotionutil.TimeString(r.CreatedAt), "name": r.Username, "imageUrl": r.ImageURL})
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Review"), Data: data, Paging: httpx.Pagination(page, limit, total)})
}

func (s *Service) update(c fiber.Ctx) error {
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	var row domain.Review
	if err := s.db.WithContext(c.Context()).First(&row, "id = ?", c.Params("id")).Error; err != nil {
		return err
	}
	if !actor.Admin && row.UserID != actor.UserID {
		return httpx.Forbidden()
	}
	if err := promotionutil.ValidateFields(c, updateFields, true); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(c.Body(), &fields); err != nil {
		return err
	}
	changes, old := map[string]any{}, map[string]any{}
	old["rating"] = row.Rating
	old["comment"] = promotionutil.NullString(row.Comment)
	if raw, ok := fields["rating"]; ok {
		var rating float64
		if err := json.Unmarshal(raw, &rating); err != nil {
			return err
		}
		if err := checkRating(rating); err != nil {
			return err
		}
		if rating != float64(int(rating)) {
			return fiber.NewError(500, "Internal server error")
		}
		changes["rating"] = int(rating)
	}
	if raw, ok := fields["comment"]; ok {
		comment, err := promotionutil.Nullable[string](raw)
		if err != nil {
			return err
		}
		changes["comment"] = promotionutil.NullString(comment)
	}
	if err := promotionutil.ValidateUpdateFields(old, changes, updateFields...); err != nil {
		return err
	}
	if err := s.db.WithContext(c.Context()).Model(&row).Updates(changes).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "Review")})
}

func (s *Service) remove(c fiber.Ctx) error {
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	var row domain.Review
	if err := s.db.WithContext(c.Context()).First(&row, "id = ?", c.Params("id")).Error; err != nil {
		return err
	}
	if !actor.Admin && row.UserID != actor.UserID {
		return httpx.Forbidden()
	}
	if err := s.db.WithContext(c.Context()).Delete(&row).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("delete", "Review")})
}
