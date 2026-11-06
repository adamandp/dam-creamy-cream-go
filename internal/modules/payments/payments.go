// Package payments preserves the current NestJS PaymentsService HTTP behavior.
// The upstream service is a scaffold: these routes do not charge, update,
// or delete payments. Checkout creates the real PENDING payment records.
package payments

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
)

type Service struct{}

func NewService() *Service { return &Service{} }

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/payments")
	r.Post("/", func(c fiber.Ctx) error { return c.Status(201).SendString("This action adds a new payment") })
	r.Get("/", func(c fiber.Ctx) error { return c.SendString("This action returns all payments") })
	r.Get("/:id", func(c fiber.Ctx) error {
		return c.SendString("This action returns a #" + legacyNumericID(c.Params("id")) + " payment")
	})
	r.Patch("/:id", func(c fiber.Ctx) error {
		return c.SendString("This action updates a #" + legacyNumericID(c.Params("id")) + " payment")
	})
	r.Delete("/:id", func(c fiber.Ctx) error {
		return c.SendString("This action removes a #" + legacyNumericID(c.Params("id")) + " payment")
	})
}

// The NestJS scaffold converts @Param('id') through JavaScript's unary +.
// Preserve the observable result (including "NaN") without touching the DB.
func legacyNumericID(raw string) string {
	if raw == "" {
		return "0"
	}
	number, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return "NaN"
	}
	return strconv.FormatFloat(number, 'f', -1, 64)
}
