package orders

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

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

func NewService(db *gorm.DB, ses *sessions.Service) *Service {
	return &Service{db: db, sessions: ses}
}
func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/orders", s.sessions.RequireActor)
	r.Post("/checkout", s.checkout)
	r.Get("/", s.sessions.RequireAdmin, s.findAll)
	r.Get("/user", s.byUser)
	r.Get("/:id", s.byID)
}

// Input keeps the unexposed NestJS CQRS checkout payload shape. Prices,
// discounts, voucher claims and product inventory are always read from the DB.
type CheckoutInput struct {
	Order struct {
		UserID     string   `json:"userId"`
		AddressID  string   `json:"addressId"`
		VoucherIDs []string `json:"voucherIds"`
	} `json:"order"`
	OrderItems []CheckoutItem `json:"orderItems"`
	Shipment   ShipmentInput  `json:"shipment"`
	Payment    struct {
		Method string `json:"method"`
	} `json:"payment"`
}

// ShipmentInput preserves the Go time.Time field used by checkout callers,
// while accepting the original NestJS z.coerce.date() JSON representations.
type ShipmentInput struct {
	Courier     string    `json:"courier"`
	Service     string    `json:"service"`
	Cost        float64   `json:"cost"`
	Estimated   time.Time `json:"estimated"`
	Description *string   `json:"description"`
}

func (in *ShipmentInput) UnmarshalJSON(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	// Zod's original pick() schema requires cost and description to be
	// present. Description may explicitly be null; cost may not.
	for _, field := range []string{"cost", "description"} {
		value, found := fields[field]
		if !found || (field == "cost" && bytes.Equal(bytes.TrimSpace(value), []byte("null"))) {
			return fmt.Errorf("shipment.%s is required", field)
		}
	}
	var wire struct {
		Courier     string          `json:"courier"`
		Service     string          `json:"service"`
		Cost        float64         `json:"cost"`
		Estimated   json.RawMessage `json:"estimated"`
		Description *string         `json:"description"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return err
	}
	estimated, err := promotionutil.ParseDate(wire.Estimated)
	if err != nil {
		return fmt.Errorf("shipment.estimated: %w", err)
	}
	*in = ShipmentInput{
		Courier:     wire.Courier,
		Service:     wire.Service,
		Cost:        wire.Cost,
		Estimated:   estimated,
		Description: wire.Description,
	}
	return nil
}

type CheckoutItem struct {
	ProductID  string  `json:"productId"`
	DiscountID *string `json:"discountId"`
	Quantity   int     `json:"quantity"`
}

func (s *Service) findAll(c fiber.Ctx) error {
	return s.list(c, "")
}
func (s *Service) byUser(c fiber.Ctx) error {
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	return s.list(c, actor.UserID)
}
func (s *Service) list(c fiber.Ctx, userID string) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	q := s.db.WithContext(c.Context()).Model(&domain.Order{})
	if userID != "" {
		q = q.Where("user_id = ?", userID)
	}
	var total int64
	if err = q.Count(&total).Error; err != nil {
		return err
	}
	data := make([]domain.Order, 0)
	if err = q.Order("created_at DESC, id DESC").Offset((page - 1) * limit).Limit(limit).Find(&data).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Orders"), Data: data, Paging: httpx.Pagination(page, limit, total)})
}
func (s *Service) byID(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	var order domain.Order
	if err := s.db.WithContext(c.Context()).First(&order, "id = ?", id).Error; err != nil {
		return err
	}
	if !actor.Admin && order.UserID != actor.UserID {
		return httpx.Forbidden()
	}
	var items []domain.OrderItem
	var payment domain.Payment
	var shipment domain.Shipment
	var voucherIDs []string
	db := s.db.WithContext(c.Context())
	if err = db.Where("order_id = ?", id).Order("created_at,id").Find(&items).Error; err != nil {
		return err
	}
	if err = db.Where("order_id = ?", id).First(&payment).Error; err != nil {
		return err
	}
	if err = db.Where("order_id = ?", id).First(&shipment).Error; err != nil {
		return err
	}
	if err = db.Model(&domain.OrderVoucher{}).Where("order_id = ?", id).Pluck("voucher_id", &voucherIDs).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "Order"), Data: fiber.Map{
		"order": order, "orderItems": items, "payment": payment, "shipment": shipment, "voucherIds": voucherIDs,
	}})
}
