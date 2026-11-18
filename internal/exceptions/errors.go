package exceptions

import (
	"dam-creamy-cream-go/internal/utils"

	"github.com/gofiber/fiber/v3"
)

func BadRequest(message string) error {
	return fiber.NewError(fiber.StatusBadRequest, message)
}

func NotFound(resource string) error {
	return fiber.NewError(fiber.StatusNotFound, utils.NotFoundMessage(resource, ""))
}

func Conflict(resource string) error {
	return fiber.NewError(fiber.StatusConflict, utils.AlreadyExistMessage(resource, ""))
}

func Unauthorized() error {
	return fiber.NewError(fiber.StatusUnauthorized, utils.NotLoggedInMessage)
}

func InternalServerError() error {
	return fiber.NewError(fiber.StatusInternalServerError, "🔥 Internal Server Error! Something went wrong on our side. Hang tight, we're fixing it! 🧯")
}
