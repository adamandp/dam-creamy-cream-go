package httpx

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type Response struct {
	StatusCode int     `json:"statusCode,omitempty"`
	Message    string  `json:"message,omitempty"`
	Data       any     `json:"data,omitempty"`
	Paging     *Paging `json:"paging,omitempty"`
}

type Paging struct {
	CurrentPage int   `json:"currentPage"`
	PageSize    int   `json:"pageSize"`
	TotalItems  int64 `json:"totalItems"`
	TotalPages  int64 `json:"totalPages"`
}

func Page(c fiber.Ctx) (int, int, error) {
	page, limit := 1, 10
	if raw := c.Query("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, fiber.NewError(400, "page must be a number")
		}
		page = n
	}
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, fiber.NewError(400, "limit must be a number")
		}
		limit = n
	}
	if page < 1 || page > 100 || limit < 1 || limit > 100 {
		return 0, 0, fiber.NewError(400, "page and limit must be between 1 and 100")
	}
	return page, limit, nil
}
func Pagination(page, limit int, total int64) *Paging {
	return &Paging{CurrentPage: page, PageSize: limit, TotalItems: total, TotalPages: (total + int64(limit) - 1) / int64(limit)}
}
func UUID(id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return fiber.NewError(400, "Invalid UUID")
	}
	return nil
}
func Bind(c fiber.Ctx, dst any) error {
	if err := c.Bind().Body(dst); err != nil {
		return fiber.NewError(400, "Invalid request body")
	}
	return nil
}

func Message(action, entity string) string {
	switch action {
	case "create":
		return fmt.Sprintf("🎉 %s created successfully! Welcome aboard! 🚀", entity)
	case "get":
		return fmt.Sprintf("📦 %s retrieved successfully! Smooth sailing! 😎", entity)
	case "update":
		return fmt.Sprintf("🔄 %s updated successfully! Looking fresh! ✨", entity)
	case "delete":
		return fmt.Sprintf("🗑️ %s deleted successfully! Sayonara! 👋", entity)
	}
	return entity
}
func NotFound(entity string) error {
	return fiber.NewError(404, fmt.Sprintf("🔍 %s not found. Maybe it teleported to another dimension? 🛸", entity))
}
func Conflict(entity string) error {
	return fiber.NewError(409, fmt.Sprintf("⚡ %s already exists! No duplicates allowed! 😆", entity))
}
func ErrorHandler(c fiber.Ctx, err error) error {
	var fe *fiber.Error
	if errors.As(err, &fe) {
		return c.Status(fe.Code).JSON(fiber.Map{"statusCode": fe.Code, "message": fe.Message, "error": statusText(fe.Code)})
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrorHandler(c, NotFound("Record"))
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		if pg.Code == "23505" {
			return ErrorHandler(c, Conflict("Record"))
		}
		if pg.Code == "23503" {
			return ErrorHandler(c, fiber.NewError(409, "Referenced record does not exist or is still in use"))
		}
	}
	return c.Status(500).JSON(fiber.Map{"statusCode": 500, "message": "Internal server error", "error": "Internal Server Error"})
}
func statusText(code int) string {
	switch code {
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 409:
		return "Conflict"
	default:
		return "Internal Server Error"
	}
}
func NonBlank(v string, min, max int, field string) error {
	n := len([]rune(strings.TrimSpace(v)))
	if n < min || n > max {
		return fiber.NewError(400, fmt.Sprintf("%s must be %d-%d characters", field, min, max))
	}
	return nil
}
