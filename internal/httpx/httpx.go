package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"sort"
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

func BindStrict(c fiber.Ctx, dst any) error {
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fiber.NewError(400, "Invalid request body: "+err.Error())
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return fiber.NewError(400, "Invalid request body: trailing content")
	} else if !errors.Is(err, io.EOF) {
		return fiber.NewError(400, "Invalid request body: "+err.Error())
	}
	return nil
}
func ExplicitNull(c fiber.Ctx, key string) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal(c.Body(), &m) != nil {
		return false
	}
	v, ok := m[key]
	return ok && bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}
func Required(c fiber.Ctx, keys ...string) error {
	var m map[string]json.RawMessage
	if json.Unmarshal(c.Body(), &m) != nil {
		return fiber.NewError(400, "Invalid request body")
	}
	for _, k := range keys {
		_, ok := m[k]
		if !ok {
			return fiber.NewError(400, k+" is required")
		}
	}
	return nil
}

var onlyLetters = regexp.MustCompile("^[A-Za-z\\s]+$")
var phonePattern = regexp.MustCompile("^[0-9]{10,15}$")
var emailPattern = regexp.MustCompile("^[A-Za-z0-9._%+\\-]+@[A-Za-z0-9.\\-]+\\.[a-zA-Z]{2,}$")

func Name(v string, min, max int, field string) error {
	if err := NonBlank(v, min, max, field); err != nil {
		return err
	}
	if !onlyLetters.MatchString(v) {
		return fiber.NewError(400, "Just letters & spaces, no secret codes! 🤨 {Name Only Letters}")
	}
	return nil
}
func Phone(v string) error {
	if err := NonBlank(v, 4, 100, "Phone number"); err != nil {
		return err
	}
	if !phonePattern.MatchString(v) {
		return fiber.NewError(400, "📞 Invalid phone number! Double-check and try again! 🤳 {Invalid Phone}")
	}
	return nil
}
func Email(v string) error {
	if err := NonBlank(v, 4, 100, "Email"); err != nil {
		return err
	}
	if !emailPattern.MatchString(v) {
		return fiber.NewError(400, "📧 Invalid email format! Try something like user@example.com ✉️ {Invalid Email}")
	}
	return nil
}
func ValidateUpdate(old, changes map[string]any) error {
	diff := 0
	same := make([]string, 0)
	for k, v := range changes {
		if reflect.DeepEqual(old[k], v) {
			same = append(same, k)
		} else {
			diff++
		}
	}
	if diff == 0 {
		return fiber.NewError(400, "📭 No changes detected. Everything is already up to date! 👍")
	}
	if len(same) > 0 {
		sort.Strings(same)
		return fiber.NewError(400, "📭 No changes detected in "+strings.Join(same, ", ")+" fields. Everything’s already up to date!")
	}
	return nil
}

func RejectNull(c fiber.Ctx, keys ...string) error {
	for _, key := range keys {
		if ExplicitNull(c, key) {
			return fiber.NewError(400, key+" must not be null")
		}
	}
	return nil
}
