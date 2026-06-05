package router

import (
	"github.com/gofiber/fiber/v2"

	"shortenerapi/internal/handler"
)

func SetupRoutes(
	app *fiber.App,
	linkHandler *handler.LinkHandler,
	analyticsHandler *handler.AnalyticsHandler,
	redirectHandler *handler.RedirectHandler,
	authHandler *handler.AuthHandler,
	authMiddleware fiber.Handler,
	rateLimitMiddleware fiber.Handler,
	redirectRateLimitMiddleware fiber.Handler,
) {
	// Health check (public)
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "ShortenerAPI",
		})
	})

	// Public redirect (with per-IP rate limit)
	app.Get("/:shortCode", redirectRateLimitMiddleware, redirectHandler.Redirect)
	app.Post("/:shortCode/unlock", redirectHandler.Unlock)

	// API Group
	api := app.Group("/api/v1")

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)

	// API Key routes (requires auth + rate limit)
	authKeys := auth.Group("/keys", authMiddleware, rateLimitMiddleware)
	authKeys.Get("", authHandler.ListAPIKeys)
	authKeys.Post("", authHandler.CreateAPIKey)
	authKeys.Delete("", authHandler.DeleteAPIKey)

	// Custom Domain routes (requires auth + rate limit)
	authDomains := auth.Group("/domains", authMiddleware, rateLimitMiddleware)
	authDomains.Get("", authHandler.ListCustomDomains)
	authDomains.Post("", authHandler.AddCustomDomain)
	authDomains.Delete("", authHandler.DeleteCustomDomain)


	// Link routes (requires auth + rate limit)
	links := api.Group("/links", authMiddleware, rateLimitMiddleware)
	links.Post("", linkHandler.Shorten)
	links.Post("/bulk", linkHandler.BulkShorten)
	links.Get("", linkHandler.List)
	links.Get("/:id", linkHandler.Get)
	links.Put("/:id", linkHandler.Update)
	links.Delete("/:id", linkHandler.Delete)

	// Analytics routes (requires auth + rate limit)
	links.Get("/:id/analytics", analyticsHandler.GetAnalytics)
	links.Get("/:id/analytics/geo", analyticsHandler.GetGeoAnalytics)
	links.Get("/:id/analytics/devices", analyticsHandler.GetDeviceAnalytics)
	links.Get("/:id/analytics/referrers", analyticsHandler.GetReferrerAnalytics)
	links.Get("/:id/analytics/timeseries", analyticsHandler.GetTimeSeriesAnalytics)
}
