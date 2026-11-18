package products

import (
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

	response, err := h.service.Create(c.Context(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(types.WebResponse[*CreateResponse]{
		Message: utils.CreateMessage(resourceName),
		Data:    response,
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

func (h *Handler) FindCatalog(c fiber.Ctx) error {
	var req FindCatalogRequest

	if err := c.Bind().Query(&req); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	if err := h.validate.Struct(&req); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	response, err := h.service.FindCatalog(c.Context(), req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

func (h *Handler) PresignImage(c fiber.Ctx) error {
	id := c.Params("id")

	if err := h.validate.Var(id, "required,uuid"); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	var req PresignImageRequest

	if err := c.Bind().JSON(&req); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	if err := h.validate.Struct(&req); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	response, err := h.service.PresignImage(c.Context(), id, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(types.WebResponse[*PresignImageResponse]{
		Message: "Presigned upload URL generated successfully",
		Data:    response,
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

	response, err := h.service.Update(c.Context(), id, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusOK).JSON(types.WebResponse[*UpdateResponse]{
		Message: utils.UpdateMessage(resourceName),
		Data:    response,
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
