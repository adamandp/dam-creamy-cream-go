package promotionutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/gofiber/fiber/v3"
)

var namePattern = regexp.MustCompile("^[A-Za-z\\s]+$")

// Name follows the public API's alphabetic/spaces rule without silently trimming input.
func Name(value, field string, min, max int) error {
	length := len([]rune(value))
	if length < min || length > max || !namePattern.MatchString(value) {
		return fiber.NewError(400, fmt.Sprintf("Invalid %s", field))
	}
	return nil
}

func Text(value, field string, min, max int) error {
	length := len([]rune(value))
	if length < min || length > max {
		return fiber.NewError(400, fmt.Sprintf("%s must be %d-%d characters", field, min, max))
	}
	return nil
}

func Enum(value, field string, options ...string) error {
	for _, option := range options {
		if value == option {
			return nil
		}
	}
	return fiber.NewError(400, fmt.Sprintf("Invalid %s", field))
}

// Date accepts JSON date strings and JavaScript epoch-millisecond values.
type Date struct{ time.Time }

func (d *Date) UnmarshalJSON(raw []byte) error {
	parsed, err := ParseDate(raw)
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}

func ParseDate(raw []byte) (time.Time, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return time.Time{}, err
	}
	switch value := v.(type) {
	case nil:
		// z.coerce.date(null) -> new Date(0).
		return time.UnixMilli(0).UTC(), nil
	case bool:
		// JavaScript Date(Boolean) treats false as 0 and true as 1 ms.
		if value {
			return time.UnixMilli(1).UTC(), nil
		}
		return time.UnixMilli(0).UTC(), nil
	case float64:
		if math.IsNaN(value) || math.IsInf(value, 0) || value > 8640000000000000 || value < -8640000000000000 {
			return time.Time{}, fmt.Errorf("date out of range")
		}
		return time.UnixMilli(int64(value)).UTC(), nil
	case string:
		if value == "" {
			return time.Time{}, fmt.Errorf("invalid date")
		}
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02", "2006-01-02T15:04:05", time.RFC1123, time.RFC1123Z} {
			if parsed, err := time.Parse(layout, value); err == nil {
				return parsed.UTC(), nil
			}
		}
		if epoch, err := strconv.ParseInt(value, 10, 64); err == nil && len(value) >= 12 {
			return time.UnixMilli(epoch).UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date")
}

func TimeString(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// BodyFields exposes only known DTO members. The API DTO parser strips
// unrecognized fields by default; the Go handlers do the same.
func BodyFields(c fiber.Ctx, allowed ...string) (map[string]json.RawMessage, error) {
	var body map[string]json.RawMessage
	if err := json.Unmarshal(c.Body(), &body); err != nil || body == nil {
		return nil, fiber.NewError(400, "Invalid request body")
	}
	allow := make(map[string]bool, len(allowed))
	for _, field := range allowed {
		allow[field] = true
	}
	for field := range body {
		if !allow[field] {
			delete(body, field)
		}
	}
	return body, nil
}

// Nullable handles either valid typed JSON or an explicit null. It never
// interprets a missing field as a null update.
func Nullable[T any](raw json.RawMessage) (*T, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, nil
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fiber.NewError(400, "Invalid request body")
	}
	return &value, nil
}

func Value[T any](raw json.RawMessage) (T, error) {
	var value T
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return value, fiber.NewError(400, "Invalid request body")
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return value, fiber.NewError(400, "Invalid request body")
	}
	return value, nil
}

func Required(c fiber.Ctx, fields ...string) error {
	return httpx.Required(c, fields...)
}

func OptionalDate(raw json.RawMessage) (time.Time, error) {
	parsed, err := ParseDate(raw)
	if err != nil {
		return time.Time{}, fiber.NewError(400, "Invalid date")
	}
	return parsed, nil
}

func NullableUUID(raw json.RawMessage, field string) (any, error) {
	val, err := Nullable[string](raw)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}
	if err := httpx.UUID(*val); err != nil {
		return nil, fiber.NewError(400, fmt.Sprintf("Invalid %s", field))
	}
	return *val, nil
}

func NullString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
func NullFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
func NullInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func AsView(model any) (map[string]any, error) {
	body, err := json.Marshal(model)
	if err != nil {
		return nil, err
	}
	var record map[string]any
	if err := json.Unmarshal(body, &record); err != nil {
		return nil, err
	}
	for key, value := range record {
		if !strings.HasSuffix(key, "At") && !strings.HasSuffix(key, "Date") {
			continue
		}
		asText, ok := value.(string)
		if !ok {
			continue
		}
		parsed, err := time.Parse(time.RFC3339Nano, asText)
		if err == nil {
			record[key] = TimeString(parsed)
		}
	}
	return record, nil
}

func AsViews[T any](models []T) ([]map[string]any, error) {
	items := make([]map[string]any, 0, len(models))
	for _, model := range models {
		view, err := AsView(model)
		if err != nil {
			return nil, err
		}
		items = append(items, view)
	}
	return items, nil
}
