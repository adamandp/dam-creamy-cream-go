package unit_test

import (
	"net/http/httptest"
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/app"
	"github.com/adamandp/dam-creamy-cream-go/internal/config"
	"github.com/gofiber/fiber/v3"
)

func TestCredentialedCORSAllowlist(t *testing.T) {
	server := fiber.New()
	app.ConfigureCORS(server, config.Config{CORSAllowedOrigins: []string{"https://store.example.com"}})
	server.Get("/ping", func(c fiber.Ctx) error { return c.SendStatus(200) })

	cases := []struct {
		method, origin string
		status         int
		allowed        bool
	}{
		{"GET", "https://store.example.com", 200, true},
		{"OPTIONS", "https://store.example.com", 204, true},
		{"GET", "https://attacker.example.com", 200, false},
		{"OPTIONS", "https://attacker.example.com", 403, false},
		{"GET", "", 200, false},
	}
	for _, tc := range cases {
		t.Run(tc.method+"-"+tc.origin, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, "/ping", nil)
			if tc.origin != "" {
				request.Header.Set("Origin", tc.origin)
			}
			response, err := server.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != tc.status {
				t.Fatalf("status=%d want=%d", response.StatusCode, tc.status)
			}
			actualOrigin := response.Header.Get("Access-Control-Allow-Origin")
			if tc.allowed && (actualOrigin != tc.origin || response.Header.Get("Access-Control-Allow-Credentials") != "true") {
				t.Fatalf("allowed origin missing CORS headers: %+v", response.Header)
			}
			if !tc.allowed && (actualOrigin != "" || response.Header.Get("Access-Control-Allow-Credentials") != "") {
				t.Fatalf("untrusted origin received credentialed CORS headers: %+v", response.Header)
			}
		})
	}
}

func TestProductionCORSRequiresExplicitOrigins(t *testing.T) {
	t.Setenv("NODE_ENV", "production")
	t.Setenv("DATABASE_URL", "postgresql://fixture:fixture@localhost:5432/fixture")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("ACCESS_JWT_SECRET", "unit-access")
	t.Setenv("REFRESH_JWT_SECRET", "unit-refresh")
	t.Setenv("MINIO_ENDPOINT", "localhost:9000")
	t.Setenv("MINIO_ACCESS_KEY", "fixture")
	t.Setenv("MINIO_SECRET_KEY", "fixture")
	t.Setenv("PUBLIC_API_URL", "https://api.example.com")

	for _, candidate := range []string{"", "*", "https://store.example.com/path", "null"} {
		t.Setenv("FRONTEND_ORIGINS", candidate)
		if _, err := config.Load(); err == nil {
			t.Errorf("origin list %q must not pass production config", candidate)
		}
	}
	t.Setenv("FRONTEND_ORIGINS", "https://store.example.com,https://admin.example.com")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.OriginAllowed("https://store.example.com") || !cfg.OriginAllowed("https://admin.example.com") || cfg.OriginAllowed("https://attacker.example.com") {
		t.Fatal("origin allowlist not enforced")
	}
}
