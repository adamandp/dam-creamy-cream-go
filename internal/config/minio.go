package config

import (
	"strings"

	"github.com/spf13/viper"
)

// MinIOConfig holds object storage connection settings.
type MinIOConfig struct {
	Endpoint  string `validate:"required"`
	AccessKey string `validate:"required"`
	SecretKey string `validate:"required"`
	Bucket    string `validate:"required"`
	UseSSL    bool
}

func loadMinIO(v *viper.Viper) MinIOConfig {
	// default local minio settings
	v.SetDefault("MINIO_ENDPOINT", "localhost:9000")
	v.SetDefault("MINIO_USE_SSL", false)

	return MinIOConfig{
		Endpoint:  strings.TrimSpace(v.GetString("MINIO_ENDPOINT")),
		AccessKey: strings.TrimSpace(v.GetString("MINIO_ACCESS_KEY")),
		SecretKey: v.GetString("MINIO_SECRET_KEY"),
		Bucket:    strings.TrimSpace(v.GetString("MINIO_BUCKET")),
		UseSSL:    v.GetBool("MINIO_USE_SSL"),
	}
}
