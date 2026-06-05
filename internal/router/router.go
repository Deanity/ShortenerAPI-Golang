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
) {
	// Health check (public)
	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "ShortenerAPI",
		})
	})

	// Public redirect
	app.Get("/:shortCode", redirectHandler.Redirect)
	app.Post("/:shortCode/unlock", redirectHandler.Unlock)

	// API Group
	api := app.Group("/api/v1")

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)

	// API Key routes (requires auth)
	authKeys := auth.Group("/keys", authMiddleware)
	authKeys.Get("", authHandler.ListAPIKeys)
	authKeys.Post("", authHandler.CreateAPIKey)
	authKeys.Delete("", authHandler.DeleteAPIKey)

	// Link routes (requires auth)
	links := api.Group("/links", authMiddleware)
	links.Post("", linkHandler.Shorten)
	links.Post("/bulk", linkHandler.BulkShorten)
	links.Get("", linkHandler.List)
	links.Get("/:id", linkHandler.Get)
	links.Put("/:id", linkHandler.Update)
	links.Delete("/:id", linkHandler.Delete)

	// Analytics routes (requires auth)
	links.Get("/:id/analytics", analyticsHandler.GetAnalytics)
}
