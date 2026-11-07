package config

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port              string
	Environment       string
	DatabaseURL       string
	RedisURL          string
	AccessJWTSecret   string
	RefreshJWTSecret  string
	MinIOEndpoint     string
	MinIOAccessKey    string
	MinIOSecretKey    string
	MinIOBucket       string
	MinIOUseSSL       bool
	PublicAPIURL      string
	RajaOngkirBaseURL string
	RajaOngkirAPIKey  string

	RajaOngkirRateLimit int

	CORSAllowedOrigins []string
}

func Load() (Config, error) {
	origins := os.Getenv("FRONTEND_ORIGINS")
	if origins == "" && env("NODE_ENV", "development") != "production" {
		origins = "http://localhost:3000,http://localhost:3001,http://localhost:5173"
	}
	allowedOrigins, err := parseOrigins(origins)
	if err != nil {
		return Config{}, err
	}
	limit, err := strconv.Atoi(env("RAJAONGKIR_RATE_LIMIT_PER_MINUTE", "30"))
	if err != nil || limit < 1 || limit > 1000 {
		return Config{}, errors.New("RAJAONGKIR_RATE_LIMIT_PER_MINUTE must be an integer between 1 and 1000")
	}
	port := env("PORT", "3000")
	ssl, err := strconv.ParseBool(env("MINIO_USE_SSL", "false"))
	if err != nil {
		return Config{}, errors.New("MINIO_USE_SSL must be true or false")
	}
	cfg := Config{
		Port:              port,
		Environment:       env("NODE_ENV", "development"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		RedisURL:          os.Getenv("REDIS_URL"),
		AccessJWTSecret:   os.Getenv("ACCESS_JWT_SECRET"),
		RefreshJWTSecret:  os.Getenv("REFRESH_JWT_SECRET"),
		MinIOEndpoint:     os.Getenv("MINIO_ENDPOINT"),
		MinIOAccessKey:    os.Getenv("MINIO_ACCESS_KEY"),
		MinIOSecretKey:    os.Getenv("MINIO_SECRET_KEY"),
		MinIOBucket:       env("MINIO_BUCKET", "creamy-cream"),
		MinIOUseSSL:       ssl,
		PublicAPIURL:      env("PUBLIC_API_URL", "http://localhost:"+port),
		RajaOngkirBaseURL: os.Getenv("RAJAONGKIR_BASE_URL"),
		RajaOngkirAPIKey:  os.Getenv("RAJAONGKIR_API_KEY"),

		RajaOngkirRateLimit: limit,

		CORSAllowedOrigins: allowedOrigins,
	}
	if cfg.DatabaseURL == "" || cfg.RedisURL == "" || cfg.AccessJWTSecret == "" || cfg.RefreshJWTSecret == "" {
		return Config{}, errors.New("DATABASE_URL, REDIS_URL, ACCESS_JWT_SECRET and REFRESH_JWT_SECRET are required")
	}
	if cfg.MinIOEndpoint == "" || cfg.MinIOAccessKey == "" || cfg.MinIOSecretKey == "" {
		return Config{}, errors.New("MINIO_ENDPOINT, MINIO_ACCESS_KEY and MINIO_SECRET_KEY are required")
	}
	if (cfg.RajaOngkirBaseURL == "") != (cfg.RajaOngkirAPIKey == "") {
		return Config{}, errors.New("RAJAONGKIR_BASE_URL and RAJAONGKIR_API_KEY must be configured together")
	}
	if cfg.Environment == "production" {
		if cfg.RajaOngkirBaseURL != "" && !strings.HasPrefix(cfg.RajaOngkirBaseURL, "https://") {
			return Config{}, errors.New("RAJAONGKIR_BASE_URL must use HTTPS in production")
		}
		if len(cfg.CORSAllowedOrigins) == 0 {
			return Config{}, errors.New("FRONTEND_ORIGINS must explicitly list trusted origins in production")
		}
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

func parseOrigins(raw string) ([]string, error) {
	origins := []string{}
	seen := map[string]bool{}
	for _, entry := range strings.Split(raw, ",") {
		origin := strings.TrimSpace(entry)
		if origin == "" {
			continue
		}
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
			parsed.Host == "" || parsed.User != nil || parsed.Path != "" ||
			parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
			return nil, errors.New("FRONTEND_ORIGINS must contain comma-separated http(s) origins without paths or wildcards")
		}
		key := strings.ToLower(origin)
		if !seen[key] {
			origins = append(origins, origin)
			seen[key] = true
		}
	}
	return origins, nil
}

// OriginAllowed compares the full scheme, host and port, never wildcard-matching
// credentialed browser requests.
func (c Config) OriginAllowed(origin string) bool {
	for _, allowed := range c.CORSAllowedOrigins {
		if strings.EqualFold(origin, allowed) {
			return true
		}
	}
	return false
}
