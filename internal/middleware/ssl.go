package middleware

import (
	"github.com/gofiber/fiber/v2"
)

// NewSSLEnforcementMiddleware redirects HTTP requests to HTTPS (except in development env)
// and sets the Strict-Transport-Security (HSTS) header.
func NewSSLEnforcementMiddleware(appEnv string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// If protocol is HTTP and app is not in development, redirect to HTTPS
		if appEnv != "development" && c.Protocol() == "http" {
			hstsURI := "https://" + c.Hostname() + c.OriginalURL()
			return c.Redirect(hstsURI, fiber.StatusMovedPermanently)
		}

		// Apply HSTS header for HTTPS traffic (or all traffic in production/non-dev)
		if c.Protocol() == "https" || appEnv != "development" {
			c.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		return c.Next()
	}
}
