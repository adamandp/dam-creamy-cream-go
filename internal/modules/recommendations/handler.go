package recommendations

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

func (h *Handler) GetClassic(c fiber.Ctx) error {
	response, err := h.service.GetClassic(c.Context())
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *Handler) GetCategories(c fiber.Ctx) error {
	response, err := h.service.GetCategories(c.Context())
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *Handler) GetOffers(c fiber.Ctx) error {
	response, err := h.service.GetOffers(c.Context())
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *Handler) GetRelated(c fiber.Ctx) error {
	id := c.Params("id")

	if err := h.validate.Var(id, "required,uuid"); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	response, err := h.service.GetRelated(c.Context(), id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
