package app

import (
	"strings"

	"github.com/adamandp/dam-creamy-cream-go/internal/config"
	"github.com/gofiber/fiber/v3"
)

// ConfigureCORS allows credentialed requests only from explicitly trusted
// frontends. Unlisted origins never receive access-control response headers.
func ConfigureCORS(httpApp *fiber.App, cfg config.Config) {
	httpApp.Use(func(c fiber.Ctx) error {
		origin := c.Get("Origin")
		if origin != "" {
			c.Set("Vary", "Origin")
			if !cfg.OriginAllowed(origin) {
				if strings.EqualFold(c.Method(), "OPTIONS") {
					return c.SendStatus(403)
				}
				return c.Next()
			}
			c.Set("Access-Control-Allow-Origin", origin)
			c.Set("Access-Control-Allow-Credentials", "true")
			c.Set("Access-Control-Allow-Headers", "Content-Type, Accept, Authorization")
			c.Set("Access-Control-Allow-Methods", "GET,HEAD,PUT,PATCH,POST,DELETE,OPTIONS")
		}
		if strings.EqualFold(c.Method(), "OPTIONS") {
			return c.SendStatus(204)
		}
		return c.Next()
	})
}
