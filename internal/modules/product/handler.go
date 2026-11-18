package product

import (
	"dam-creamy-cream-go/internal/exceptions"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service  *Service
	validate *validator.Validate
}

func NewHandler(service *Service, validate *validator.Validate) *Handler {
	return &Handler{
		service:  service,
		validate: validate,
	}
}

func (h *Handler) FindOne(c fiber.Ctx) error {
	id := c.Params("id")

	if err := h.validate.Var(id, "required,uuid"); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	response, err := h.service.FindOne(c.Context(), id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *Handler) FindInformation(c fiber.Ctx) error {
	id := c.Params("id")

	if err := h.validate.Var(id, "required,uuid"); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	response, err := h.service.FindInformation(c.Context(), id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
