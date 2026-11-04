package health

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
)

type Service struct {
	db     *sql.DB
	client *http.Client
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db, client: &http.Client{Timeout: 3 * time.Second}}
}
func (s *Service) Routes(app fiber.Router) {
	app.Get("/health", s.web)
	app.Get("/health/db", s.postgres)
}
func (s *Service) web(c fiber.Ctx) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://google.com", nil)
	if err != nil {
		return err
	}
	res, err := s.client.Do(req)
	if err != nil {
		return c.Status(503).JSON(fiber.Map{"status": "error", "details": fiber.Map{"google": fiber.Map{"status": "down"}}})
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return c.Status(503).JSON(fiber.Map{"status": "error", "details": fiber.Map{"google": fiber.Map{"status": "down"}}})
	}
	return c.JSON(fiber.Map{"status": "ok", "info": fiber.Map{"google": fiber.Map{"status": "up"}}, "error": fiber.Map{}, "details": fiber.Map{"google": fiber.Map{"status": "up"}}})
}
func (s *Service) postgres(c fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.db.PingContext(ctx); err != nil {
		return c.Status(503).JSON(fiber.Map{"status": "error",
			"info":    fiber.Map{},
			"error":   fiber.Map{"db": fiber.Map{"status": "down"}},
			"details": fiber.Map{"db": fiber.Map{"status": "down"}},
		})
	}
	return c.JSON(fiber.Map{"status": "ok",
		"info":    fiber.Map{"db": fiber.Map{"status": "up"}},
		"error":   fiber.Map{},
		"details": fiber.Map{"db": fiber.Map{"status": "up"}},
	})
}
