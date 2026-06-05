package main

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"shortenerapi/pkg/config"
	"shortenerapi/pkg/database"
)

func main() {
	log.Info().Msg("Starting database seeder...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := database.ConnectMongo(cfg.MongoURI)
	if err != nil {
		log.Fatal().Err(err).Msg("MongoDB connection failed, cannot seed database")
	}
	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Error().Err(err).Msg("Error disconnecting MongoDB client")
		}
	}()

	_ = mongoClient.Database(cfg.MongoDBName)

	log.Info().Msg("Database seeded successfully")
}
