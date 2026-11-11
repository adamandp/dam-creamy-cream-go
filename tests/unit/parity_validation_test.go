package unit_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/gofiber/fiber/v3"
)

func TestNestValidationRules(t *testing.T) {
	for _, tt := range []struct {
		value string
		valid bool
	}{
		{"Jane Doe", true}, {"John123", false}, {"ab", true}, {"", false},
	} {
		err := httpx.Name(tt.value, 1, 100, "Name")
		if (err == nil) != tt.valid {
			t.Errorf("name %q should be valid=%t; got %v", tt.value, tt.valid, err)
		}
	}
	for _, tt := range []struct {
		value string
		valid bool
	}{
		{"john@example.com", true}, {"john-at-example", false}, {"someone@no", false},
	} {
		err := httpx.Email(tt.value)
		if (err == nil) != tt.valid {
			t.Errorf("email %q should be valid=%t; got %v", tt.value, tt.valid, err)
		}
	}
	for _, tt := range []struct {
		value string
		valid bool
	}{
		{"081234567890", true}, {"hello", false}, {"123456", false},
	} {
		err := httpx.Phone(tt.value)
		if (err == nil) != tt.valid {
			t.Errorf("phone %q should be valid=%t; got %v", tt.value, tt.valid, err)
		}
	}
}

func TestNestStringLengthUsesUTF16CodeUnits(t *testing.T) {
	for _, tc := range []struct {
		text string
		want int
	}{
		{"vanilla", 7},
		{"中", 1},
		{"🍦", 2},
		{"🍦🍦", 4},
	} {
		if got := httpx.JSStringLength(tc.text); got != tc.want {
			t.Errorf("JSStringLength(%q)=%d; want %d", tc.text, got, tc.want)
		}
	}
	if err := httpx.NonBlank("🍦", 2, 2, "description"); err != nil {
		t.Fatalf("two UTF-16 code units must be accepted: %v", err)
	}
	if err := httpx.NonBlank("🍦", 1, 1, "description"); err == nil {
		t.Fatal("one UTF-16 code unit limit must reject an emoji")
	}
}
func TestUpdateGuard(t *testing.T) {
	old := map[string]any{"name": "Jane", "is_active": true}
	for _, tt := range []struct {
		changes map[string]any
		bad     bool
	}{
		{map[string]any{}, true},
		{map[string]any{"name": "Jane"}, true},
		{map[string]any{"name": "Jane", "is_active": false}, true},
		{map[string]any{"name": "Janet"}, false},
	} {
		err := httpx.ValidateUpdate(old, tt.changes)
		if (err != nil) != tt.bad {
			t.Errorf("changes=%v got err=%v", tt.changes, err)
		}
	}
}
func TestStrictBodyAndRequiredMember(t *testing.T) {
	app := fiber.New()
	app.Post("/check", func(c fiber.Ctx) error {
		var data struct{ Name string }
		if err := httpx.BindStrict(c, &data); err != nil {
			return err
		}
		if err := httpx.Required(c, "Name"); err != nil {
			return err
		}
		return c.SendStatus(200)
	})
	for _, tt := range []struct {
		body   string
		status int
	}{
		{`{"Name":"Jane"}`, 200},
		{`{"Name":"Jane","unexpected":true}`, 400},
		{`{}`, 400},
	} {
		req := httptest.NewRequest("POST", "/check", strings.NewReader(tt.body))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != tt.status {
			t.Errorf("body %s got %d want %d", tt.body, resp.StatusCode, tt.status)
		}
	}
}
