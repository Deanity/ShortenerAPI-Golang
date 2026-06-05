package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"shortenerapi/internal/domain"
	"shortenerapi/pkg/cache"
	"shortenerapi/pkg/utils"
)

// RateLimitConfig holds per-plan request limits (requests per minute).
type RateLimitConfig struct {
	FreePlanLimit  int
	ProPlanLimit   int
	RedirectLimit  int // Per IP for redirect endpoint
}

// NewRateLimitMiddleware returns a middleware that enforces per-key rate limits via Redis.
// It reads the user from c.Locals("user") set by the auth middleware.
func NewRateLimitMiddleware(linkCache *cache.LinkCache, cfg RateLimitConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := c.Locals("user").(*domain.User)
		if !ok {
			return c.Next()
		}

		limit := cfg.FreePlanLimit
		if user.Plan == "pro" || user.Plan == "enterprise" {
			limit = cfg.ProPlanLimit
		}

		identifier := fmt.Sprintf("user:%s", user.ID.Hex())
		count, err := linkCache.IncrRateLimit(c.Context(), identifier)
		if err != nil {
			return c.Next() // fail open if Redis is down
		}

		if int(count) > limit {
			c.Set("Retry-After", "60")
			return utils.Error(c, fiber.StatusTooManyRequests,
				fmt.Sprintf("Rate limit exceeded. Max %d requests/minute for your plan.", limit),
				"RATE_LIMIT_EXCEEDED",
			)
		}

		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", limit-int(count)))
		return c.Next()
	}
}

// NewRedirectRateLimitMiddleware enforces per-IP rate limiting for the redirect endpoint.
func NewRedirectRateLimitMiddleware(linkCache *cache.LinkCache, limitPerMinute int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		identifier := fmt.Sprintf("ip:%s", c.IP())
		count, err := linkCache.IncrRateLimit(c.Context(), identifier)
		if err != nil {
			return c.Next()
		}

		if int(count) > limitPerMinute {
			c.Set("Retry-After", "60")
			return utils.Error(c, fiber.StatusTooManyRequests, "Too many requests", "RATE_LIMIT_EXCEEDED")
		}
		return c.Next()
	}
}
