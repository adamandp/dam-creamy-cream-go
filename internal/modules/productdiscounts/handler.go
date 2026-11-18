package productdiscounts

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
	return &Handler{
		service:  service,
		validate: validate,
	}
}

func (h *Handler) Create(c fiber.Ctx) error {
	var req CreateRequest

	if err := c.Bind().JSON(&req); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	if err := h.validate.Struct(&req); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	if err := h.service.Create(c.Context(), req); err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(types.MessageResponse{
		Message: utils.CreateMessage(resourceName),
	})
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

	productDiscount, err := h.service.FindByID(c.Context(), id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(types.WebResponse[*model.ProductDiscount]{
		Message: utils.GetMessage(resourceName),
		Data:    productDiscount,
	})
}

func (h *Handler) FindByProductID(c fiber.Ctx) error {
	productID := c.Params("productId")

	if err := h.validate.Var(productID, "required,uuid"); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	data, err := h.service.FindByProductID(c.Context(), productID)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(types.WebResponse[*[]*model.ProductDiscount]{
		Message: utils.GetMessage(resourceName),
		Data:    &data,
	})
}

func (h *Handler) Delete(c fiber.Ctx) error {
	id := c.Params("id")

	if err := h.validate.Var(id, "required,uuid"); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	if err := h.service.Delete(c.Context(), id); err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(types.MessageResponse{
		Message: utils.DeleteMessage(resourceName),
	})
}
