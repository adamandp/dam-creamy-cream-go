package payments

import (
	"github.com/gofiber/fiber/v3"
)

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
