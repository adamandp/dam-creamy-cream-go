package promotionutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/gofiber/fiber/v3"
	"reflect"
	"regexp"
	"sort"
	"strings"
)

// Field mirrors the relevant Zod schema constraints.
type Field struct {
	Key, Label, Kind string
	Min, Max         int
	Nullable         bool
	Values           []string
}

var (
	uuidPattern = "/^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}|00000000-0000-0000-0000-000000000000|ffffffff-ffff-ffff-ffff-ffffffffffff)$/"
	uuidZod     = regexp.MustCompile("^([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}|00000000-0000-0000-0000-000000000000|ffffffff-ffff-ffff-ffff-ffffffffffff)$")
)

func issue(path, code, message string) map[string]any {
	return map[string]any{"path": []string{path}, "code": code, "message": message}
}
func invalidType(f Field, expected string) map[string]any {
	msg := "Invalid input: expected " + expected
	switch f.Kind {
	case "number":
		msg = fmt.Sprintf("🔢 %s must be a valid number. No funky math here! 🚫 {Invalid Number}", f.Label)
	case "boolean":
		msg = fmt.Sprintf("⚖️ %s must be either true or false. No Schrödinger values! 🐱 {Invalid Boolean}", f.Label)
	case "string", "name":
		msg = fmt.Sprintf("📜 %s must be a valid text. No weird symbols! 🚫 {Invalid String}", f.Label)
	case "date":
		msg = fmt.Sprintf("📅 %s must be a valid date. Time travel not allowed! ⏳ {Invalid Date}", f.Label)
	}
	i := issue(f.Key, "invalid_type", msg)
	i["expected"] = expected
	if f.Kind == "date" {
		i["received"] = "Invalid Date"
	}
	return i
}
func lengthIssue(f Field, tooSmall bool) map[string]any {
	code, key, bound, msg := "too_big", "maximum", f.Max, fmt.Sprintf("🚀 Whoa! %s is too long. Maximum allowed is %d characters! ✍️ {Maximum Length}", f.Label, f.Max)
	if tooSmall {
		code, key, bound, msg = "too_small", "minimum", f.Min, fmt.Sprintf("🔡 %s must be at least %d characters. Keep going! 💪 {Minimum Length}", f.Label, f.Min)
	}
	i := issue(f.Key, code, msg)
	i["origin"], i[key], i["inclusive"] = "string", bound, true
	return i
}
func zodFieldIssues(f Field, raw json.RawMessage, exists bool) []map[string]any {
	if exists && bytes.Equal(bytes.TrimSpace(raw), []byte("null")) && f.Nullable {
		return nil
	}
	if f.Kind == "enum" {
		var v string
		if exists && json.Unmarshal(raw, &v) == nil {
			for _, opt := range f.Values {
				if v == opt {
					return nil
				}
			}
		}
		i := issue(f.Key, "invalid_value", fmt.Sprintf("🎭 %s must be one of the following enum. Choose wisely! 🎯 {Invalid Enum}", f.Label))
		i["values"] = f.Values
		return []map[string]any{i}
	}
	if f.Kind == "date" {
		if exists {
			if _, err := ParseDate(raw); err == nil {
				return nil
			}
		}
		return []map[string]any{invalidType(f, "date")}
	}
	if f.Kind == "uuid" {
		if !exists {
			return []map[string]any{{"expected": "string", "code": "invalid_type", "path": []string{f.Key}, "message": "Invalid input: expected string, received undefined"}}
		}
		var v any
		if json.Unmarshal(raw, &v) != nil {
			return []map[string]any{invalidType(f, "string")}
		}
		str, ok := v.(string)
		if !ok {
			kind := "null"
			switch v.(type) {
			case float64:
				kind = "number"
			case bool:
				kind = "boolean"
			case []any:
				kind = "array"
			case map[string]any:
				kind = "object"
			}
			return []map[string]any{{"expected": "string", "code": "invalid_type", "path": []string{f.Key}, "message": "Invalid input: expected string, received " + kind}}
		}
		if uuidZod.MatchString(str) {
			return nil
		}
		i := issue(f.Key, "invalid_format", fmt.Sprintf("🆔 %s must be a valid UUID. No random gibberish! 🔢 {Invalid UUID}", f.Label))
		i["origin"], i["format"], i["pattern"] = "string", "uuid", uuidPattern
		return []map[string]any{i}
	}
	var v any
	if !exists || json.Unmarshal(raw, &v) != nil {
		expected := f.Kind
		if expected == "name" {
			expected = "string"
		}
		return []map[string]any{invalidType(f, expected)}
	}
	switch f.Kind {
	case "number":
		if _, ok := v.(float64); !ok {
			return []map[string]any{invalidType(f, "number")}
		}
	case "boolean":
		if _, ok := v.(bool); !ok {
			return []map[string]any{invalidType(f, "boolean")}
		}
	case "string", "name":
		str, ok := v.(string)
		if !ok {
			return []map[string]any{invalidType(f, "string")}
		}
		issues := []map[string]any{}
		n := len([]rune(str))
		if n < f.Min {
			issues = append(issues, lengthIssue(f, true))
		}
		if f.Max > 0 && n > f.Max {
			issues = append(issues, lengthIssue(f, false))
		}
		if f.Kind == "name" && !namePattern.MatchString(str) {
			i := issue(f.Key, "invalid_format", "Just letters & spaces, no secret codes! 🤨 {Name Only Letters}")
			i["origin"], i["format"], i["pattern"] = "string", "regex", "/^[A-Za-z\\s]+$/"
			issues = append(issues, i)
		}
		return issues
	}
	return nil
}

// ValidateFields reproduces NestJS ZodValidationPipe JSON errors.
func ValidateFields(c fiber.Ctx, fields []Field, partial bool) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(c.Body(), &raw); err != nil || raw == nil {
		return &httpx.ValidationError{Issues: []map[string]any{{"expected": "object", "code": "invalid_type", "path": []string{}, "message": "Invalid input: expected object, received null"}}}
	}
	issues := []map[string]any{}
	for _, f := range fields {
		v, exists := raw[f.Key]
		if !exists && partial {
			continue
		}
		issues = append(issues, zodFieldIssues(f, v, exists)...)
	}
	if len(issues) > 0 {
		return &httpx.ValidationError{Issues: issues}
	}
	return nil
}

func DiscountFields() []Field {
	return []Field{
		{Key: "name", Label: "Discount Name", Kind: "name", Min: 1, Max: 100},
		{Key: "discountType", Label: "Discount Type", Kind: "enum", Values: []string{"PERCENTAGE", "FIXED", "FREE_ITEM", "PRICE"}},
		{Key: "value", Label: "Value", Kind: "number", Nullable: true},
		{Key: "maxApplied", Label: "Max Applied", Kind: "number"},
		{Key: "maxRedeemed", Label: "Max Reedem", Kind: "number"},
		{Key: "redeemedCount", Label: "Redeemed Count", Kind: "number"},
		{Key: "description", Label: "Description", Kind: "string", Min: 1, Max: 255, Nullable: true},
		{Key: "productId", Label: "Product ID", Kind: "uuid", Nullable: true},
		{Key: "quantity", Label: "Quantity", Kind: "number", Nullable: true},
		{Key: "startDate", Label: "Start Date", Kind: "date"},
		{Key: "endDate", Label: "End Date", Kind: "date"},
	}
}
func VoucherFields() []Field {
	return []Field{
		{Key: "code", Label: "Code", Kind: "string", Min: 1, Max: 50},
		{Key: "name", Label: "Name", Kind: "name", Min: 1, Max: 100},
		{Key: "description", Label: "Description", Kind: "string", Min: 1, Max: 255, Nullable: true},
		{Key: "type", Label: "Discount Type", Kind: "enum", Values: []string{"PERCENTAGE", "FIXED", "FREE_ITEM"}},
		{Key: "group", Label: "Group", Kind: "enum", Values: []string{"CHECKOUT", "SHIPPING"}},
		{Key: "value", Label: "Value", Kind: "number", Nullable: true},
		{Key: "productId", Label: "Product ID", Kind: "uuid", Nullable: true},
		{Key: "quantity", Label: "Quantity", Kind: "number", Nullable: true},
		{Key: "minimumPurchase", Label: "Minimum Price", Kind: "number"},
		{Key: "isActive", Label: "Is Active", Kind: "boolean"},
		{Key: "startDate", Label: "Start Date", Kind: "date"},
		{Key: "endDate", Label: "End Date", Kind: "date"},
		{Key: "maxRedeemed", Label: "Max Redeemed", Kind: "number"},
		{Key: "redeemedCount", Label: "Redeemed Count", Kind: "number"},
	}
}
func UserVoucherFields() []Field {
	return []Field{
		{Key: "userId", Label: "User ID", Kind: "uuid"},
		{Key: "voucherId", Label: "Voucher ID", Kind: "uuid"},
		{Key: "claimedAt", Label: "Claimed At", Kind: "date"},
		{Key: "expiredAt", Label: "Expired At", Kind: "date"},
	}
}
func ProductDiscountFields() []Field {
	return []Field{{Key: "productId", Label: "Product ID", Kind: "uuid"}, {Key: "discountId", Label: "Discount ID", Kind: "uuid"}}
}

// Preserve public camelCase in no-change messages. Zod creates distinct JS
// Date objects, so submitting even an equal date is treated as a change.
func ValidateUpdateFields(existing, updated map[string]any, schema ...Field) error {
	camel := func(s string) string {
		p := strings.Split(s, "_")
		for i := 1; i < len(p); i++ {
			if len(p[i]) > 0 {
				p[i] = strings.ToUpper(p[i][:1]) + p[i][1:]
			}
		}
		return strings.Join(p, "")
	}
	old, changes := map[string]any{}, map[string]any{}
	for k, v := range existing {
		old[camel(k)] = v
	}
	for k, v := range updated {
		if k == "start_date" || k == "end_date" {
			old[camel(k)] = nil
		}
		changes[camel(k)] = v
	}
	// Zod returns keys in the schema declaration order, not in map/sorted
	// order. Match the original validateUpdate same-fields message.
	keys := make([]string, 0, len(changes))
	used := map[string]bool{}
	for _, field := range schema {
		if _, ok := changes[field.Key]; ok {
			keys = append(keys, field.Key)
			used[field.Key] = true
		}
	}
	others := []string{}
	for key := range changes {
		if !used[key] {
			others = append(others, key)
		}
	}
	sort.Strings(others)
	keys = append(keys, others...)
	same := []string{}
	changed := 0
	for _, key := range keys {
		if reflect.DeepEqual(old[key], changes[key]) {
			same = append(same, key)
		} else {
			changed++
		}
	}
	if changed == 0 {
		return fiber.NewError(400, "📭 No changes detected. Everything is already up to date! 👍")
	}
	if len(same) > 0 {
		return fiber.NewError(400, "📭 No changes detected in "+strings.Join(same, ", ")+" fields. Everything’s already up to date!")
	}
	return nil
}
