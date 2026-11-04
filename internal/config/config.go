package config

import (
	"errors"
	"os"
)

type Config struct {
	Port                string
	Environment         string
	DatabaseURL         string
	RedisURL            string
	AccessJWTSecret     string
	RefreshJWTSecret    string
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
}

func Load() (Config, error) {
	cfg := Config{
		Port:                env("PORT", "3000"),
		Environment:         env("NODE_ENV", "development"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		RedisURL:            os.Getenv("REDIS_URL"),
		AccessJWTSecret:     os.Getenv("ACCESS_JWT_SECRET"),
		RefreshJWTSecret:    os.Getenv("REFRESH_JWT_SECRET"),
		CloudinaryCloudName: os.Getenv("CLOUDINARY_CLOUD_NAME"),
		CloudinaryAPIKey:    os.Getenv("CLOUDINARY_API_KEY"),
		CloudinaryAPISecret: os.Getenv("CLOUDINARY_API_SECRET"),
	}
	if cfg.DatabaseURL == "" || cfg.RedisURL == "" || cfg.AccessJWTSecret == "" || cfg.RefreshJWTSecret == "" {
		return Config{}, errors.New("DATABASE_URL, REDIS_URL, ACCESS_JWT_SECRET and REFRESH_JWT_SECRET are required")
	}
	return cfg, nil
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
