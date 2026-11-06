package orders

import (
	"math"
	"strings"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

func badInput(reason string) error    { return fiber.NewError(400, reason) }
func unavailable(reason string) error { return fiber.NewError(409, reason) }
func saneMoney(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1e12
}
func amount(value float64) float64 { return math.Round(value*100) / 100 }
func validateCheckout(b CheckoutInput, who sessions.Actor) error {
	if err := httpx.UUID(b.Order.UserID); err != nil {
		return err
	}
	if b.Order.UserID != who.UserID {
		return httpx.Forbidden()
	}
	if err := httpx.UUID(b.Order.AddressID); err != nil {
		return err
	}
	if len(b.OrderItems) < 1 {
		return badInput("orderItems must contain at least one product")
	}
	if len(b.Order.VoucherIDs) > 2 {
		return badInput("at most two voucherIds are allowed")
	}
	seen := map[string]bool{}
	for _, id := range b.Order.VoucherIDs {
		if err := httpx.UUID(id); err != nil {
			return err
		}
		if seen[id] {
			return badInput("duplicate voucherId")
		}
		seen[id] = true
	}
	items := map[string]bool{}
	for _, item := range b.OrderItems {
		if err := httpx.UUID(item.ProductID); err != nil {
			return err
		}
		if item.Quantity < 1 {
			return badInput("item quantity must be positive")
		}
		if items[item.ProductID] {
			return badInput("duplicate productId in orderItems")
		}
		items[item.ProductID] = true
		if item.DiscountID != nil {
			if err := httpx.UUID(*item.DiscountID); err != nil {
				return err
			}
		}
	}
	if !saneMoney(b.Shipment.Cost) {
		return badInput("shipment.cost must be nonnegative and finite")
	}
	if len(strings.TrimSpace(b.Shipment.Courier)) < 1 || len(b.Shipment.Courier) > 50 {
		return badInput("invalid shipment.courier")
	}
	if len(strings.TrimSpace(b.Shipment.Service)) < 1 || len(b.Shipment.Service) > 50 {
		return badInput("invalid shipment.service")
	}
	if b.Shipment.Estimated.IsZero() {
		return badInput("shipment.estimated must be an RFC3339 timestamp")
	}
	if b.Shipment.Description != nil && len(*b.Shipment.Description) > 255 {
		return badInput("shipment.description too long")
	}
	if len(strings.TrimSpace(b.Payment.Method)) < 1 || len(b.Payment.Method) > 50 {
		return badInput("invalid payment.method")
	}
	return nil
}

func (s *Service) checkout(c fiber.Ctx) error {
	var body CheckoutInput
	if err := httpx.BindStrict(c, &body); err != nil {
		return err
	}
	who, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	if err := validateCheckout(body, who); err != nil {
		return err
	}
	var result CheckoutResult
	err = s.db.WithContext(c.Context()).Transaction(func(tx *gorm.DB) error {
		var innerErr error
		result, innerErr = s.checkoutTransaction(tx, body)
		return innerErr
	})
	if err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("create", "Order"), Data: result})
}

type CheckoutResult struct {
	OrderID        string  `json:"orderId"`
	TotalPrice     float64 `json:"totalPrice"`
	DiscountAmount float64 `json:"discountAmount"`
	VoucherAmount  float64 `json:"voucherAmount"`
	ShippingCost   float64 `json:"shippingCost"`
	FinalPrice     float64 `json:"finalPrice"`
	Status         string  `json:"status"`
}

// time.Now() is sampled once per checkout, so all discounts/vouchers use the
// same validity instant despite the multi-step transaction.
func sameInstant() time.Time { return time.Now().UTC() }
