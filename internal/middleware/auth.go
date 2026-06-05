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
			user, key, err := authUseCase.ValidateAPIKey(c.Context(), apiKey)
			if err != nil {
				return utils.Error(c, fiber.StatusUnauthorized, "Invalid API key", "INVALID_API_KEY")
			}

			// Enforce scoping
			requiredScope := determineRequiredScope(c.Method(), c.Path())
			if requiredScope != "" && key != nil {
				if !hasScope(key.Scopes, requiredScope) {
					return utils.Error(c, fiber.StatusForbidden, "API key has insufficient permissions", "INSUFFICIENT_PERMISSIONS")
				}
			}

			c.Locals("user", user)
			c.Locals("api_key", key)
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

func determineRequiredScope(method, path string) string {
	if strings.Contains(path, "/analytics") {
		return "analytics:read"
	}
	if strings.HasPrefix(path, "/api/v1/links") {
		if method == "GET" {
			return "links:read"
		}
		return "links:write"
	}
	return ""
}

func hasScope(scopes []string, required string) bool {
	if len(scopes) == 0 {
		return true // Default to unrestricted key (backward compatibility)
	}
	for _, s := range scopes {
		if s == "*" || s == required {
			return true
		}
	}
	return false
}

