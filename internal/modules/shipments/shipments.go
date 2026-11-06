package shipments

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type QuoteClient interface {
	Calculate(context.Context, CostInput) (ShippingQuote, error)
}

type Service struct {
	db       *gorm.DB
	sessions *sessions.Service
	quotes   QuoteClient
}

func NewService(db *gorm.DB, ses *sessions.Service, quotes QuoteClient) *Service {
	return &Service{db: db, sessions: ses, quotes: quotes}
}

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/shipments")
	// Quoting is public in NestJS. No provider credentials are sent to callers.
	r.Post("/cost", s.cost)
	r.Get("/", s.sessions.RequireAdmin, s.findAll)
	r.Get("/:id", s.sessions.RequireActor, s.findByID)
	r.Patch("/:id/status", s.sessions.RequireAdmin, s.updateStatus)
}

func (s *Service) findAll(c fiber.Ctx) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	q := s.db.WithContext(c.Context()).Model(&domain.Shipment{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return httpx.NotFound("Shipment")
	}
	data := make([]domain.Shipment, 0)
	if err := q.Order("created_at DESC, id DESC").Offset((page - 1) * limit).Limit(limit).Find(&data).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Shipment"), Data: data, Paging: httpx.Pagination(page, limit, total)})
}

func (s *Service) findByID(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	var shipment domain.Shipment
	db := s.db.WithContext(c.Context())
	if err := db.First(&shipment, "id = ?", id).Error; err != nil {
		return err
	}
	if !actor.Admin {
		var order domain.Order
		if err := db.Select("user_id").First(&order, "id = ?", shipment.OrderID).Error; err != nil {
			return err
		}
		if order.UserID != actor.UserID {
			return httpx.Forbidden()
		}
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Shipment"), Data: shipment})
}

func (s *Service) updateStatus(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var in struct {
		Status string `json:"status"`
	}
	if err := httpx.Bind(c, &in); err != nil {
		return err
	}
	if err := httpx.Required(c, "status"); err != nil {
		return err
	}
	switch in.Status {
	case "PENDING", "SHIPPED", "DELIVERED", "CANCELLED":
	default:
		return fiber.NewError(400, "Invalid shipment status")
	}
	// NestJS updates only the Shipment.status; Order/Payment state is unchanged.
	updated := s.db.WithContext(c.Context()).Model(&domain.Shipment{}).
		Where("id = ?", id).Update("status", in.Status)
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected == 0 {
		return httpx.NotFound("Shipment")
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "Shipment")})
}

type CostInput struct {
	Origin      float64
	Destination float64
	Weight      float64
}

func parseCoercedNumber(raw json.RawMessage) (float64, error) {
	if len(raw) == 0 {
		return 0, fiber.NewError(400, "Shipping origin, destination and weight are required")
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, fiber.NewError(400, "Invalid shipping number")
	}
	var number float64
	switch v := value.(type) {
	case float64:
		number = v
	case string:
		if strings.TrimSpace(v) == "" {
			number = 0
		} else {
			parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
			if err != nil {
				return 0, fiber.NewError(400, "Invalid shipping number")
			}
			number = parsed
		}
	case bool:
		if v {
			number = 1
		}
	case nil:
		number = 0
	default:
		return 0, fiber.NewError(400, "Invalid shipping number")
	}
	if math.IsNaN(number) || math.IsInf(number, 0) || number <= 0 {
		return 0, fiber.NewError(400, "Shipping numbers must be positive and finite")
	}
	return number, nil
}

func (s *Service) cost(c fiber.Ctx) error {
	var request struct {
		Origin      json.RawMessage `json:"origin"`
		Destination json.RawMessage `json:"destination"`
		Weight      json.RawMessage `json:"weight"`
	}
	if err := httpx.BindStrict(c, &request); err != nil {
		return err
	}
	origin, err := parseCoercedNumber(request.Origin)
	if err != nil {
		return err
	}
	destination, err := parseCoercedNumber(request.Destination)
	if err != nil {
		return err
	}
	weight, err := parseCoercedNumber(request.Weight)
	if err != nil {
		return err
	}
	if s.quotes == nil {
		return fiber.NewError(503, "Shipping quote provider is not configured")
	}
	quote, err := s.quotes.Calculate(c.Context(), CostInput{Origin: origin, Destination: destination, Weight: weight})
	if err != nil {
		return fiber.NewError(502, "Unable to retrieve shipping costs")
	}
	return c.Status(201).JSON(fiber.Map{"message": quote.Message, "status": quote.Status, "data": quote.Data})
}
