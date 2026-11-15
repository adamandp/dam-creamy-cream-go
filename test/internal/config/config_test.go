package config_test

import (
	"os"
	"testing"

	"github.com/go-playground/validator/v10"

	"dam-creamy-cream-go/internal/config"
)

func TestLoad(t *testing.T) {
	t.Setenv("MINIO_ACCESS_KEY", "test-access")
	t.Setenv("MINIO_SECRET_KEY", "test-secret")
	t.Setenv("MINIO_BUCKET", "test-bucket")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_NAME", "test")
	validate := validator.New(validator.WithRequiredStructEnabled())
	for _, tc := range []struct {
		name, env, port string
		invalid         bool
	}{
		{"shell only", "production", "8080", false},
		{"missing app env", "", "3000", true},
		{"blank app env", "  ", "3000", true},
		{"missing port", "development", "", true},
		{"non numeric port", "development", "abc", true},
		{"zero port", "development", "0", true},
		{"negative port", "development", "-1", true},
		{"port too large", "development", "65536", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			t.Setenv("APP_ENV", tc.env)
			t.Setenv("PORT", tc.port)
			cfg, err := config.Load(validate)
			if (err != nil) != tc.invalid {
				t.Fatalf("Load() error = %v, want invalid %v", err, tc.invalid)
			}
			if !tc.invalid && (cfg.AppEnv != "production" || cfg.Port != 8080) {
				t.Fatalf("unexpected config: %+v", cfg)
			}
		})
	}
}

func TestEnvFileAndShellOverride(t *testing.T) {
	t.Setenv("MINIO_ACCESS_KEY", "test-access")
	t.Setenv("MINIO_SECRET_KEY", "test-secret")
	t.Setenv("MINIO_BUCKET", "test-bucket")
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_NAME", "test")
	validate := validator.New(validator.WithRequiredStructEnabled())
	t.Chdir(t.TempDir())
	t.Setenv("APP_ENV", "")
	t.Setenv("PORT", "")
	if err := os.WriteFile(".env", []byte("APP_ENV=development\nPORT=3000\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(validate)
	if err != nil || cfg.AppEnv != "development" || cfg.Port != 3000 {
		t.Fatalf("file configuration = %+v, %v", cfg, err)
	}
	t.Setenv("PORT", "8080")
	cfg, err = config.Load(validate)
	if err != nil || cfg.Port != 8080 {
		t.Fatalf("shell override = %+v, %v", cfg, err)
	}
}

func TestInvalidDatabaseConfig(t *testing.T) {
	t.Setenv("MINIO_ACCESS_KEY", "test-access")
	t.Setenv("MINIO_SECRET_KEY", "test-secret")
	t.Setenv("MINIO_BUCKET", "test-bucket")
	for _, tc := range []struct{ key, value string }{
		{"DB_USER", " "},
		{"DB_NAME", " "},
		{"DB_PORT", "65536"},
		{"DB_SSLMODE", "invalid"},
	} {
		t.Run(tc.key, func(t *testing.T) {
			t.Chdir(t.TempDir())
			for key, value := range map[string]string{
				"APP_ENV":    "development",
				"PORT":       "3000",
				"DB_HOST":    "localhost",
				"DB_PORT":    "5432",
				"DB_USER":    "postgres",
				"DB_NAME":    "test",
				"DB_SSLMODE": "disable",
			} {
				t.Setenv(key, value)
			}
			t.Setenv(tc.key, tc.value)
			if _, err := config.Load(validator.New(validator.WithRequiredStructEnabled())); err == nil {
				t.Fatal("expected invalid database configuration to fail")
			}
		})
	}
}
