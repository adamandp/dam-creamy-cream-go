package products

import (
	"net/http/httptest"
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/gofiber/fiber/v3"
)

func TestCatalogRejectsInvalidFiltersBeforeQueryingPostgres(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	app.Get("/products/catalog", (&Service{}).catalog)

	for _, query := range []string{
		"min=NaN", "max=NaN", "min=Infinity", "max=-Infinity",
		"cat=not-a-uuid", "cat=123e4567-e89b-42d3-a456-426614174000,broken",
	} {
		req := httptest.NewRequest("GET", "/products/catalog?"+query, nil)
		response, err := app.Test(req)
		if err != nil {
			t.Fatalf("%s: %v", query, err)
		}
		_ = response.Body.Close()
		if response.StatusCode != 400 {
			t.Errorf("%s: HTTP %d; want 400", query, response.StatusCode)
		}
	}
}
