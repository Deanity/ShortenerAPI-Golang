package main

import (
	"context"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"shortenerapi/internal/handler"
	"shortenerapi/internal/middleware"
	"shortenerapi/internal/repository"
	"shortenerapi/internal/router"
	"shortenerapi/internal/usecase"
	"shortenerapi/pkg/cache"
	"shortenerapi/pkg/config"
	"shortenerapi/pkg/database"
)


func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("Starting ShortenerAPI server...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}
	log.Info().Str("env", cfg.AppEnv).Msg("Configuration loaded successfully")

	// Connect to MongoDB
	mongoClient, err := database.ConnectMongo(cfg.MongoURI)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to MongoDB")
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			log.Error().Err(err).Msg("Error disconnecting MongoDB client")
		}
	}()
	log.Info().Msg("Connected to MongoDB successfully")

	db := mongoClient.Database(cfg.MongoDBName)

	// Connect to Redis
	redisClient, err := database.ConnectRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer redisClient.Close()
	log.Info().Msg("Connected to Redis successfully")

	// Initialize cache layer
	linkCache := cache.NewLinkCache(redisClient)

	// Wire repositories
	linkRepo := repository.NewLinkRepository(db)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	userRepo := repository.NewUserRepository(db)

	mainHost := "localhost"
	if u, err := url.Parse(cfg.AppBaseURL); err == nil && u.Host != "" {
		mainHost = u.Hostname()
	}

	// Wire use cases
	authUseCase := usecase.NewAuthUseCase(userRepo, cfg.AppSecretKey)
	linkUseCase := usecase.NewLinkUseCase(linkRepo, analyticsRepo, userRepo, linkCache, cfg.GoogleSafeBrowsingAPIKey, mainHost)
	analyticsUseCase := usecase.NewAnalyticsUseCase(analyticsRepo, linkRepo, linkCache)

	// Wire handlers
	linkHandler := handler.NewLinkHandler(linkUseCase)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsUseCase)
	redirectHandler := handler.NewRedirectHandler(linkUseCase, analyticsUseCase, cfg.WebhookMaxRetries, cfg.WebhookTimeoutSeconds)
	authHandler := handler.NewAuthHandler(authUseCase)

	// Wire middleware
	authMiddleware := middleware.NewAuthMiddleware(authUseCase)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(linkCache, middleware.RateLimitConfig{
		FreePlanLimit: cfg.RateLimitFree,
		ProPlanLimit:  cfg.RateLimitPro,
		RedirectLimit: cfg.RateLimitRedirect,
	})
	redirectRateLimitMiddleware := middleware.NewRedirectRateLimitMiddleware(linkCache, cfg.RateLimitRedirect)

	// Initialize Fiber
	app := fiber.New(fiber.Config{
		AppName: "ShortenerAPI",
	})

	// Global SSL and HSTS enforcement middleware
	app.Use(middleware.NewSSLEnforcementMiddleware(cfg.AppEnv))


	// Setup routes
	router.SetupRoutes(
		app,
		linkHandler,
		analyticsHandler,
		redirectHandler,
		authHandler,
		authMiddleware,
		rateLimitMiddleware,
		redirectRateLimitMiddleware,
	)

	port := cfg.AppPort
	if port == "" {
		port = "8080"
	}
	log.Info().Msgf("Server listening on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
