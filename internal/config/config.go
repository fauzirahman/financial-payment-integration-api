package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort     string
	DatabaseURL string
}

func Load() (*Config, error) {
	// Load .env jika tersedia.
	// Pada environment production, environment variable
	// yang sudah tersedia tetap dapat digunakan.
	_ = godotenv.Load()

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")

	return &Config{
		AppPort:     appPort,
		DatabaseURL: databaseURL,
	}, nil
}
