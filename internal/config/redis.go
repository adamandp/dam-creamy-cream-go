package config

import (
	"strings"

	"github.com/spf13/viper"
)

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host     string `validate:"required"`
	Port     int    `validate:"required,min=1,max=65535"`
	Username string
	Password string
	DB       int `validate:"min=0"`
}

func loadRedis(v *viper.Viper) RedisConfig {
	// default local redis settings
	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", 6379)
	v.SetDefault("REDIS_DB", 0)

	return RedisConfig{
		Host:     strings.TrimSpace(v.GetString("REDIS_HOST")),
		Port:     v.GetInt("REDIS_PORT"),
		Username: strings.TrimSpace(v.GetString("REDIS_USERNAME")),
		Password: v.GetString("REDIS_PASSWORD"),
		DB:       v.GetInt("REDIS_DB"),
	}
}
