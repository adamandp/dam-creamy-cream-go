// Package config loads application settings from environment variables.
package config

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config holds validated application settings.
type Config struct {
	Database DatabaseConfig
	MinIO    MinIOConfig
	Redis    RedisConfig
	AppEnv   string `validate:"required"`
	Port     int    `validate:"required,min=1,max=65535"`
}

// Load reads the optional .env file and environment overrides, then validates them.
func Load(validate *validator.Validate) (Config, error) {
	// load .env with environment variable overrides
	v := viper.New()
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()
	_ = v.ReadInConfig()

	// read configuration values
	cfg := Config{
		Database: loadDatabase(v),
		MinIO:    loadMinIO(v),
		Redis:    loadRedis(v),
		AppEnv:   strings.TrimSpace(v.GetString("APP_ENV")),
		Port:     v.GetInt("PORT"),
	}

	// validate configuration
	if err := validate.Struct(cfg); err != nil {
		return Config{}, fmt.Errorf("validate configuration: %w", err)
	}

	return cfg, nil
}
