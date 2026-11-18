package health_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"dam-creamy-cream-go/internal/modules/health"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"gorm.io/gorm"
)

func TestHealth(t *testing.T) {
	app := fiber.New()
	app.Use(requestid.New())
	health.RegisterRoutes(app, health.Params{})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/health/go", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	if resp.Header.Get("X-Request-ID") == "" {
		t.Error("missing request ID from server middleware")
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	data, ok := body["data"].(map[string]any)
	if !ok || len(body) != 2 || body["message"] != "Application is healthy" || data["status"] != "ok" {
		t.Errorf("body = %v, want data.status:ok", body)
	}
}

// probeConnector supplies an in-memory SQL connection for health endpoint tests.
type probeConnector struct{ err error }

func (p probeConnector) Connect(context.Context) (driver.Conn, error) {
	return probeConnection{err: p.err}, nil
}
func (p probeConnector) Driver() driver.Driver { return probeDriver{} }

type probeDriver struct{}

func (probeDriver) Open(string) (driver.Conn, error) { return probeConnection{}, nil }

type probeConnection struct{ err error }

func (p probeConnection) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unsupported") }
func (p probeConnection) Begin() (driver.Tx, error)           { return nil, errors.New("unsupported") }
func (p probeConnection) Close() error                        { return nil }
func (p probeConnection) Ping(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok {
		return errors.New("missing probe deadline")
	}
	return p.err
}

func TestDatabaseHealth(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		code   int
		status string
	}{
		{"connected", nil, http.StatusOK, "ok"},
		{"unreachable", errors.New("private database error"), http.StatusServiceUnavailable, "unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := sql.OpenDB(probeConnector{err: tc.err})
			t.Cleanup(func() { _ = pool.Close() })
			client := &gorm.DB{Config: &gorm.Config{ConnPool: pool}}
			app := fiber.New()
			health.RegisterRoutes(app, health.Params{Database: client})
			response, err := app.Test(httptest.NewRequest(http.MethodGet, "/health/db", nil))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != tc.code {
				t.Fatalf("status = %d, want %d", response.StatusCode, tc.code)
			}
			if response.Header.Get("Cache-Control") != "no-store" {
				t.Error("missing no-store header")
			}
			var body map[string]any
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			data, ok := body["data"].(map[string]any)
			if !ok || data["status"] != tc.status {
				t.Fatalf("unexpected data: %v", body)
			}
			if body["message"] == nil || body["message"] == "" {
				t.Fatal("missing health message")
			}
			if tc.err != nil && body["errors"] != "Database unavailable" {
				t.Fatalf("unexpected errors: %v", body)
			}
			if tc.err == nil && len(body) != 2 {
				t.Fatalf("unexpected response: %v", body)
			}
		})
	}
}

func TestMissingDependencyHealth(t *testing.T) {
	app := fiber.New()
	health.RegisterRoutes(app, health.Params{})
	for _, name := range []string{"redis", "minio"} {
		t.Run(name, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/health/"+name, nil))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusServiceUnavailable {
				t.Fatalf("status = %d", resp.StatusCode)
			}
			var body struct {
				Errors string        `json:"errors"`
				Data   health.Result `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Errors == "" || body.Data.Status != "unavailable" {
				t.Fatalf("unexpected body: %+v", body)
			}
		})
	}
}
