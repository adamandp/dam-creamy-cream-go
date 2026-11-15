package config

import (
	"strings"

	"github.com/spf13/viper"
)

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host     string `validate:"required"`
	Port     int    `validate:"required,min=1,max=65535"`
	User     string `validate:"required"`
	Password string
	Name     string `validate:"required"`
	SSLMode  string `validate:"required,oneof=disable allow prefer require verify-ca verify-full"`
}

func loadDatabase(v *viper.Viper) DatabaseConfig {
	// default local database settings
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_SSLMODE", "disable")

	return DatabaseConfig{
		Host:     strings.TrimSpace(v.GetString("DB_HOST")),
		Port:     v.GetInt("DB_PORT"),
		User:     strings.TrimSpace(v.GetString("DB_USER")),
		Password: v.GetString("DB_PASSWORD"),
		Name:     strings.TrimSpace(v.GetString("DB_NAME")),
		SSLMode:  strings.TrimSpace(v.GetString("DB_SSLMODE")),
	}
}
