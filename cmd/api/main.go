package main

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"shortenerapi/internal/handler"
	"shortenerapi/internal/middleware"
	"shortenerapi/internal/repository"
	"shortenerapi/internal/router"
	"shortenerapi/internal/usecase"
	"shortenerapi/pkg/config"
	"shortenerapi/pkg/database"
)

func main() {
	// Initialize logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Info().Msg("Starting ShortenerAPI server...")

	// Load configuration
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
	_ = redisClient // will be used for caching & rate limiting in later phases

	// Wire repositories
	linkRepo := repository.NewLinkRepository(db)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	userRepo := repository.NewUserRepository(db)

	// Wire use cases
	authUseCase := usecase.NewAuthUseCase(userRepo, cfg.AppSecretKey)
	linkUseCase := usecase.NewLinkUseCase(linkRepo, analyticsRepo)
	analyticsUseCase := usecase.NewAnalyticsUseCase(analyticsRepo)

	// Wire handlers
	linkHandler := handler.NewLinkHandler(linkUseCase)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsUseCase)
	redirectHandler := handler.NewRedirectHandler(linkUseCase, analyticsUseCase)
	authHandler := handler.NewAuthHandler(authUseCase)

	// Auth middleware
	authMiddleware := middleware.NewAuthMiddleware(authUseCase)

	// Initialize Fiber application
	app := fiber.New(fiber.Config{
		AppName: "ShortenerAPI",
	})

	// Setup routes
	router.SetupRoutes(app, linkHandler, analyticsHandler, redirectHandler, authHandler, authMiddleware)

	// Start server
	port := cfg.AppPort
	if port == "" {
		port = "8080"
	}
	log.Info().Msgf("Server listening on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
