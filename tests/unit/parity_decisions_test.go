package unit_test

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
	"github.com/gofiber/fiber/v3"
)

func TestApprovedStrictUUIDAndPrismaIntegerCoercion(t *testing.T) {
	valid := "55555555-5555-4555-8555-555555555555"
	if err := httpx.UUID(valid); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"{" + valid + "}", "urn:uuid:" + valid, strings.ReplaceAll(valid, "-", ""), "invalid-id"} {
		err := httpx.UUID(v)
		if err == nil {
			t.Errorf("URI parser accepted noncanonical UUID %q", v)
		}
		var fiberErr *fiber.Error
		if !errors.As(err, &fiberErr) || fiberErr.Code != 400 || fiberErr.Message != "Validation failed (uuid is expected)" {
			t.Errorf("invalid UUID response: %q: %v", v, err)
		}
	}
	if got := promotionutil.Integer(1.5); got != 1 {
		t.Errorf("Prisma Int 1.5 -> %d, wanted 1", got)
	}
	if got := promotionutil.Integer(-1.9); got != -1 {
		t.Errorf("Prisma Int -1.9 -> %d, wanted -1", got)
	}
	n := 2.5
	if v := promotionutil.NullableInteger(&n); v == nil || *v != 2 {
		t.Errorf("nullable int: %v", v)
	}
	if promotionutil.NullableInteger(nil) != nil {
		t.Error("null should remain null")
	}
}

func TestZodPaginationResponseAndCoercion(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	app.Get("/check", func(c fiber.Ctx) error {
		page, limit, err := httpx.Page(c)
		if err != nil {
			return err
		}
		return c.JSON(fiber.Map{"page": page, "limit": limit})
	})
	for _, q := range []struct {
		path         string
		code         int
		page         int
		limit        int
		invalidField string
		issueCode    string
	}{
		{"/check?page=1.0&limit=1e1", 200, 1, 10, "", ""},
		{"/check?page=1e0&limit=10.0", 200, 1, 10, "", ""},
		{"/check?page=abc", 400, 0, 0, "page", "invalid_type"},
		{"/check?limit=0", 400, 0, 0, "limit", "too_small"},
		{"/check?page=101", 400, 0, 0, "page", "too_big"},
	} {
		req := httptest.NewRequest("GET", q.path, nil)
		response, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]any
		err = json.NewDecoder(response.Body).Decode(&result)
		_ = response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != q.code {
			t.Fatalf("%s: got %d, wanted %d; %+v", q.path, response.StatusCode, q.code, result)
		}
		if q.code == 200 {
			if result["page"] != float64(q.page) || result["limit"] != float64(q.limit) {
				t.Errorf("%s: pagination fields: %+v", q.path, result)
			}
		} else {
			errs, ok := result["errors"].([]any)
			if !ok || len(errs) != 1 {
				t.Fatalf("%s: missing Zod issues: %+v", q.path, result)
			}
			item := errs[0].(map[string]any)
			path := item["path"].([]any)
			if item["code"] != q.issueCode || path[0] != q.invalidField || result["message"] != "Validation failed" {
				t.Errorf("%s: invalid Zod issue: %+v", q.path, result)
			}
		}
	}
}

func TestPromotionZodFieldsAndDateNoopParity(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	app.Post("/check", func(c fiber.Ctx) error {
		if err := promotionutil.ValidateFields(c, promotionutil.DiscountFields(), false); err != nil {
			return err
		}
		return c.SendStatus(201)
	})
	req := httptest.NewRequest("POST", "/check", strings.NewReader(
		`{"name":"Bad123","discountType":"INVALID","value":null,"maxApplied":"not a number","maxRedeemed":2,"redeemedCount":0,"description":null,"productId":null,"quantity":null,"startDate":"2026-01-01","endDate":"2027-01-01"}`))
	req.Header.Set("Content-Type", "application/json")
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	issues := result["errors"].([]any)
	if response.StatusCode != 400 || len(issues) != 3 {
		t.Fatalf("expected 3 original Zod issues, got %d: %+v", response.StatusCode, result)
	}
	if issues[0].(map[string]any)["code"] != "invalid_format" || issues[1].(map[string]any)["code"] != "invalid_value" || issues[2].(map[string]any)["code"] != "invalid_type" {
		t.Errorf("unexpected Zod issue sequence: %+v", issues)
	}
	date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := promotionutil.ValidateUpdateFields(map[string]any{"start_date": date}, map[string]any{"start_date": date}); err != nil {
		t.Fatalf("the NestJS Date-object comparison allows an equivalent timestamp: %v", err)
	}
	if err := promotionutil.ValidateUpdateFields(map[string]any{"name": "Old", "max_applied": 3}, map[string]any{"name": "New", "max_applied": 3}); err == nil || !strings.Contains(err.Error(), "maxApplied") {
		t.Errorf("no-change message must use the DTO's camelCase field, got %v", err)
	}
	v, err := promotionutil.ParseDate([]byte(`"0"`))
	if err != nil {
		t.Fatal(err)
	}
	expected := time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local).UTC()
	if !v.Equal(expected) {
		t.Errorf("Date('0') must use JavaScript coercion: %v vs %v", v, expected)
	}
}
