package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"dam-creamy-cream-go/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	Client *gorm.DB
	pool   *sql.DB
}

func NewDatabase(ctx context.Context, cfg config.DatabaseConfig) (*Database, error) {
	connectionURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Path:   "/" + cfg.Name,
	}

	query := connectionURL.Query()
	query.Set("sslmode", cfg.SSLMode)
	query.Set("connect_timeout", "5")
	connectionURL.RawQuery = query.Encode()

	db, err := gorm.Open(
		postgres.Open(connectionURL.String()),
		&gorm.Config{
			DisableAutomaticPing: true,
			Logger: logger.Default.LogMode(
				logger.Error,
			),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	pool, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get database pool: %w", err)
	}

	pool.SetMaxOpenConns(10)
	pool.SetMaxIdleConns(5)
	pool.SetConnMaxLifetime(30 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.PingContext(pingCtx); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &Database{
		Client: db,
		pool:   pool,
	}, nil
}

func (db *Database) Close() error {
	return db.pool.Close()
}
