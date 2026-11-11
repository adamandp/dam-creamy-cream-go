package carts

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/gofiber/fiber/v3"
)

func validateInput(in Input) error {
	if err := httpx.UUID(in.ProductID); err != nil {
		return err
	}
	if in.Quantity < 1 || in.Quantity > 100 {
		return fiber.NewError(400, "quantity must be between 1 and 100")
	}
	return nil
}
