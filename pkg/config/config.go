package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	AppEnv                    string `mapstructure:"APP_ENV"`
	AppPort                   string `mapstructure:"APP_PORT"`
	AppBaseURL                string `mapstructure:"APP_BASE_URL"`
	AppSecretKey              string `mapstructure:"APP_SECRET_KEY"`
	MongoURI                  string `mapstructure:"MONGO_URI"`
	MongoDBName               string `mapstructure:"MONGO_DB_NAME"`
	RedisAddr                 string `mapstructure:"REDIS_ADDR"`
	RedisPassword             string `mapstructure:"REDIS_PASSWORD"`
	RedisDB                   int    `mapstructure:"REDIS_DB"`
	RateLimitFree             int    `mapstructure:"RATE_LIMIT_FREE"`
	RateLimitPro              int    `mapstructure:"RATE_LIMIT_PRO"`
	RateLimitRedirect         int    `mapstructure:"RATE_LIMIT_REDIRECT"`
	GoogleSafeBrowsingAPIKey  string `mapstructure:"GOOGLE_SAFE_BROWSING_API_KEY"`
	WebhookTimeoutSeconds     int    `mapstructure:"WEBHOOK_TIMEOUT_SECONDS"`
	WebhookMaxRetries         int    `mapstructure:"WEBHOOK_MAX_RETRIES"`
	LogLevel                  string `mapstructure:"LOG_LEVEL"`
	LogFormat                 string `mapstructure:"LOG_FORMAT"`
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("MONGO_URI", "mongodb://localhost:27017")
	viper.SetDefault("MONGO_DB_NAME", "shortener_api")
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("REDIS_DB", 0)

	if err := viper.ReadInConfig(); err != nil {
		// If .env doesn't exist, we can fallback to env vars or ignore it
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
