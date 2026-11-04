package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
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

// ValidationError matches the original NestJS ZodValidationPipe JSON contract.
type ValidationError struct {
	Issues []map[string]any
}

func (e *ValidationError) Error() string  { return "Validation failed" }
func ZodIssue(issue map[string]any) error { return &ValidationError{Issues: []map[string]any{issue}} }

func invalidNumber(field, label string) error {
	return ZodIssue(map[string]any{
		"code": "invalid_type", "expected": "number", "received": "NaN",
		"message": fmt.Sprintf("🔢 %s must be a valid number. No funky math here! 🚫 {Invalid Number}", label),
		"path":    []string{field},
	})
}
func boundNumber(field, label string, bound int, tooSmall bool) error {
	code, key, msg := "too_big", "maximum", "🚀 Whoa! %s is too long. Maximum allowed is %d characters! ✍️ {Maximum Length}"
	if tooSmall {
		code, key, msg = "too_small", "minimum", "🔡 %s must be at least %d characters. Keep going! 💪 {Minimum Length}"
	}
	return ZodIssue(map[string]any{
		"code": code, "origin": "number", key: bound, "inclusive": true,
		"message": fmt.Sprintf(msg, label, bound), "path": []string{field},
	})
}
func Page(c fiber.Ctx) (int, int, error) {
	parse := func(field, label string, defaultValue int) (int, error) {
		raw := c.Query(field)
		if raw == "" && len(c.Request().URI().QueryArgs().Peek(field)) == 0 {
			// Absent parameter uses the schema default. Empty supplied values
			// are handled below as Zod's coerce.number("") -> 0.
			present := false
			c.Request().URI().QueryArgs().VisitAll(func(k, _ []byte) {
				if string(k) == field {
					present = true
				}
			})
			if !present {
				return defaultValue, nil
			}
		}
		n, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if raw == "" {
			n, err = 0, nil
		}
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
			return 0, invalidNumber(field, label)
		}
		if n < 1 {
			return 0, boundNumber(field, label, 1, true)
		}
		if n > 100 {
			return 0, boundNumber(field, label, 100, false)
		}
		// The Prisma findMany skip/take arguments are integers. Zod will
		// accept decimal notations of whole numbers such as 1.0 or 1e0.
		if math.Trunc(n) != n {
			return 0, invalidNumber(field, label)
		}
		return int(n), nil
	}
	page, err := parse("page", "Page", 1)
	if err != nil {
		return 0, 0, err
	}
	limit, err := parse("limit", "Limit", 10)
	if err != nil {
		return 0, 0, err
	}
	return page, limit, nil
}
func Pagination(page, limit int, total int64) *Paging {
	return &Paging{CurrentPage: page, PageSize: limit, TotalItems: total, TotalPages: (total + int64(limit) - 1) / int64(limit)}
}

var canonicalUUID = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[1-8][a-fA-F0-9]{3}-[89aAbB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$`)

func UUID(id string) error {
	// google/uuid.Parse also accepts urn:, curly braces and unhyphenated
	// values; NestJS ParseUUIDPipe rejects those. Validate the wire shape.
	if !canonicalUUID.MatchString(id) {
		return fiber.NewError(400, "Validation failed (uuid is expected)")
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
	var validation *ValidationError
	if errors.As(err, &validation) {
		return c.Status(400).JSON(fiber.Map{"statusCode": 400, "message": "Validation failed", "errors": validation.Issues})
	}
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

func Unauthorized() error {
	return fiber.NewError(401, "🚪 You are not logged in. Please sign in to continue! 🔐")
}
func Forbidden() error {
	return fiber.NewError(403, "Forbidden resource")
}
