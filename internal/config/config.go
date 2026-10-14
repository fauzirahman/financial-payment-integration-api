package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort       string
	DatabaseURL   string
	WebhookSecret string
	AuthTokenSecret string
}

func Load() (*Config, error) {
	// Load .env jika tersedia.
	// Pada environment production, environment variable
	// yang sudah tersedia tetap dapat digunakan.
	_ = godotenv.Load()

	appPort := os.Getenv("PORT")
	if appPort == "" {
		appPort = os.Getenv("APP_PORT")
	}
	if appPort == "" {
		appPort = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	webhookSecret := os.Getenv("WEBHOOK_SECRET")
	authTokenSecret := os.Getenv("AUTH_TOKEN_SECRET")
	if len(strings.TrimSpace(authTokenSecret)) < 32 {
		return nil, fmt.Errorf("AUTH_TOKEN_SECRET must be set and contain at least 32 characters")
	}

	return &Config{
		AppPort:         appPort,
		DatabaseURL:     databaseURL,
		WebhookSecret:   webhookSecret,
		AuthTokenSecret: authTokenSecret,
	}, nil
}
