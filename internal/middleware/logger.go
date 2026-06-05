package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

func NewStructuredLogger(log zerolog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Log structured info using zerolog
		return c.Next()
	}
}
