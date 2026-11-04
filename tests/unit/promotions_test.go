package unit_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
	"github.com/gofiber/fiber/v3"
)

func TestPromotionDateCoercion(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want string
	}{
		{"iso", `"2026-11-04T15:22:30.120Z"`, "2026-11-04T15:22:30.120Z"},
		{"short", `"2026-01-01"`, "2026-01-01T00:00:00.000Z"},
		{"milliseconds", `1762344000000`, "2025-11-05T12:00:00.000Z"},
		{"null coerces zero", `null`, "1970-01-01T00:00:00.000Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := promotionutil.ParseDate([]byte(tc.raw))
			if err != nil {
				t.Fatal(err)
			}
			if got := promotionutil.TimeString(parsed); got != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
	for _, raw := range []string{`"tomorrow"`, `"not a date"`, `[]`, `{}`} {
		if _, err := promotionutil.ParseDate([]byte(raw)); err == nil {
			t.Errorf("invalid date %s accepted", raw)
		}
	}
}

func TestPromotionValidators(t *testing.T) {
	if err := promotionutil.Name("Summer Deal", "Discount Name", 1, 100); err != nil {
		t.Fatal(err)
	}
	if err := promotionutil.Name("Promo50", "Discount Name", 1, 100); err == nil {
		t.Fatal("digits in name should be rejected")
	}
	if err := promotionutil.Text("WELCOME_20", "Code", 1, 50); err != nil {
		t.Fatal(err)
	}
	if err := promotionutil.Enum("FREE_ITEM", "Discount Type", "FREE_ITEM", "FIXED"); err != nil {
		t.Fatal(err)
	}
	if err := promotionutil.Enum("PRICE", "Voucher Type", "FREE_ITEM", "FIXED"); err == nil {
		t.Fatal("invalid voucher type accepted")
	}
	for _, raw := range []string{`null`, `2`, `[]`} {
		if _, err := promotionutil.Value[string]([]byte(raw)); err == nil {
			t.Errorf("non-string %s accepted", raw)
		}
	}
	value, err := promotionutil.Nullable[float64]([]byte(`null`))
	if err != nil || value != nil {
		t.Errorf("nullable float: %v %v", value, err)
	}
}

func TestPromotionFieldSelection(t *testing.T) {
	app := fiber.New()
	app.Patch("/check", func(c fiber.Ctx) error {
		fields, err := promotionutil.BodyFields(c, "name", "value")
		if err != nil {
			return err
		}
		if _, ok := fields["ignored"]; ok {
			return fiber.NewError(500, "unexpected field forwarded")
		}
		if len(fields) != 2 {
			return fiber.NewError(400, "incorrect selected fields")
		}
		return c.SendStatus(200)
	})
	body := `{"name":"Summer Deal","value":null,"ignored":"never persisted"}`
	req := httptest.NewRequest("PATCH", "/check", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("unknown field should be discarded, status=%d", res.StatusCode)
	}
}

func TestPromotionDateOutput(t *testing.T) {
	input := map[string]any{
		"createdAt":     time.Date(2026, 11, 4, 15, 0, 0, 0, time.UTC),
		"redeemedCount": 0,
		"startDate":     time.Date(2026, 11, 4, 15, 0, 0, 120000000, time.UTC),
	}
	view, err := promotionutil.AsView(input)
	if err != nil {
		t.Fatal(err)
	}
	if view["createdAt"] != "2026-11-04T15:00:00.000Z" || view["startDate"] != "2026-11-04T15:00:00.120Z" {
		t.Fatalf("dates formatted incorrectly: %v", view)
	}
	_, err = json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
}
