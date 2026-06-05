package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"shortenerapi/internal/domain"
	"shortenerapi/pkg/utils"
)

// NewAuthMiddleware validates JWT Bearer token or X-API-Key header.
// On success, stores *domain.User in c.Locals("user").
func NewAuthMiddleware(authUseCase domain.AuthUseCase) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try X-API-Key first
		if apiKey := c.Get("X-API-Key"); apiKey != "" {
			user, err := authUseCase.ValidateAPIKey(c.Context(), apiKey)
			if err != nil {
				return utils.Error(c, fiber.StatusUnauthorized, "Invalid API key", "INVALID_API_KEY")
			}
			c.Locals("user", user)
			return c.Next()
		}

		// Try Bearer token
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return utils.Error(c, fiber.StatusUnauthorized, "Authentication required", "MISSING_AUTH")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return utils.Error(c, fiber.StatusUnauthorized, "Invalid authorization header format", "INVALID_AUTH_FORMAT")
		}

		user, err := authUseCase.ValidateToken(c.Context(), parts[1])
		if err != nil {
			return utils.Error(c, fiber.StatusUnauthorized, "Invalid or expired token", "INVALID_TOKEN")
		}

		c.Locals("user", user)
		return c.Next()
	}
}
