package inventory

import (
	"dam-creamy-cream-go/internal/dal/model"
	"dam-creamy-cream-go/internal/exceptions"
	"dam-creamy-cream-go/internal/types"
	"dam-creamy-cream-go/internal/utils"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	service  *Service
	validate *validator.Validate
}

func NewHandler(service *Service, validate *validator.Validate) *Handler {
	return &Handler{service: service, validate: validate}
}

func (h *Handler) FindAll(c fiber.Ctx) error {
	var req FindAllRequest

	if err := c.Bind().Query(&req); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	if err := h.validate.Struct(&req); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	response, err := h.service.FindAll(c.Context(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *Handler) FindByID(c fiber.Ctx) error {
	id := c.Params("id")

	if err := h.validate.Var(id, "required,uuid"); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	inventory, err := h.service.FindByID(c.Context(), id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(types.WebResponse[*model.Inventory]{
		Message: utils.GetMessage(resourceName),
		Data:    inventory,
	})
}

func (h *Handler) FindByProductID(c fiber.Ctx) error {
	id := c.Params("id")

	if err := h.validate.Var(id, "required,uuid"); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	inventory, err := h.service.FindByProductID(c.Context(), id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(types.WebResponse[*model.Inventory]{
		Message: utils.GetMessage(resourceName),
		Data:    inventory,
	})
}

func (h *Handler) Update(c fiber.Ctx) error {
	id := c.Params("id")

	if err := h.validate.Var(id, "required,uuid"); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	var req UpdateRequest

	if err := c.Bind().JSON(&req); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	if err := h.validate.Struct(&req); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	if err := h.service.Update(c.Context(), id, req); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(types.MessageResponse{
		Message: utils.UpdateMessage(resourceName),
	})
}
