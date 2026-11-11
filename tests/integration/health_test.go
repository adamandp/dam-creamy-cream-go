package integration_test

import (
	"net/http/httptest"
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/modules/health"
	"github.com/gofiber/fiber/v3"
)

func TestHealthDBReturnsNestTerminusEnvelope(t *testing.T) {
	db := connectPrismaFixture(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	health.NewService(sqlDB).Routes(app)
	result, err := app.Test(httptest.NewRequest("GET", "/health/db", nil))
	if err != nil {
		t.Fatal(err)
	}
	payload := testBody(t, result)
	if result.StatusCode != 200 || payload["status"] != "ok" {
		t.Fatalf("status not equivalent to Nest Terminus: %v", payload)
	}
	info, ok := payload["info"].(map[string]any)
	if !ok {
		t.Fatalf("missing info in health response: %v", payload)
	}
	postgres, ok := info["db"].(map[string]any)
	if !ok || postgres["status"] != "up" {
		t.Fatalf("unexpected db status: %v", payload)
	}
	if _, ok := payload["details"]; !ok {
		t.Errorf("missing details key: %v", payload)
	}
	if _, ok := payload["error"]; !ok {
		t.Errorf("missing error key: %v", payload)
	}
}
