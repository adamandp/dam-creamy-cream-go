package config

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	Port             string
	Environment      string
	DatabaseURL      string
	RedisURL         string
	AccessJWTSecret  string
	RefreshJWTSecret string
	MinIOEndpoint    string
	MinIOAccessKey   string
	MinIOSecretKey   string
	MinIOBucket      string
	MinIOUseSSL      bool
	PublicAPIURL     string
}

func Load() (Config, error) {
	port := env("PORT", "3000")
	ssl, err := strconv.ParseBool(env("MINIO_USE_SSL", "false"))
	if err != nil {
		return Config{}, errors.New("MINIO_USE_SSL must be true or false")
	}
	cfg := Config{
		Port:             port,
		Environment:      env("NODE_ENV", "development"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		RedisURL:         os.Getenv("REDIS_URL"),
		AccessJWTSecret:  os.Getenv("ACCESS_JWT_SECRET"),
		RefreshJWTSecret: os.Getenv("REFRESH_JWT_SECRET"),
		MinIOEndpoint:    os.Getenv("MINIO_ENDPOINT"),
		MinIOAccessKey:   os.Getenv("MINIO_ACCESS_KEY"),
		MinIOSecretKey:   os.Getenv("MINIO_SECRET_KEY"),
		MinIOBucket:      env("MINIO_BUCKET", "creamy-cream"),
		MinIOUseSSL:      ssl,
		PublicAPIURL:     env("PUBLIC_API_URL", "http://localhost:"+port),
	}
	if cfg.DatabaseURL == "" || cfg.RedisURL == "" || cfg.AccessJWTSecret == "" || cfg.RefreshJWTSecret == "" {
		return Config{}, errors.New("DATABASE_URL, REDIS_URL, ACCESS_JWT_SECRET and REFRESH_JWT_SECRET are required")
	}
	if cfg.MinIOEndpoint == "" || cfg.MinIOAccessKey == "" || cfg.MinIOSecretKey == "" {
		return Config{}, errors.New("MINIO_ENDPOINT, MINIO_ACCESS_KEY and MINIO_SECRET_KEY are required")
	}
	if cfg.Environment == "production" {
		if os.Getenv("PUBLIC_API_URL") == "" {
			return Config{}, errors.New("PUBLIC_API_URL must be explicitly configured in production")
		}
		if len(cfg.PublicAPIURL) < len("https://") || cfg.PublicAPIURL[:len("https://")] != "https://" {
			return Config{}, errors.New("PUBLIC_API_URL must use HTTPS in production")
		}
	}
	return cfg, nil
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
