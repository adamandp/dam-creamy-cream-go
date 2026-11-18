package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"dam-creamy-cream-go/internal/config"
	"dam-creamy-cream-go/internal/modules/categories"
	"dam-creamy-cream-go/internal/modules/health"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/sirupsen/logrus"
)

type Application struct {
	Server  *fiber.App
	cleanup []func() error
}

func Bootstrap(
	ctx context.Context,
	cfg config.Config,
	log *logrus.Entry,
	validate *validator.Validate,
) (application *Application, err error) {
	appState := &Application{}

	// clean up initialized dependencies if bootstrap fails
	defer func() {
		if err != nil {
			err = errors.Join(err, appState.Close())
		}
	}()

	// initialize database
	database, err := NewDatabase(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("initialize database: %w", err)
	}
	appState.cleanup = append(appState.cleanup, database.Close)

	// migrate models using the existing PostgreSQL connection
	if err := Migrate(ctx, database.Client); err != nil {
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	// initialize minio
	minioClient, err := NewMinIO(ctx, cfg.MinIO, log)
	if err != nil {
		return nil, fmt.Errorf("initialize minio: %w", err)
	}

	// initialize redis
	redisClient, err := NewRedis(ctx, cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("initialize redis: %w", err)
	}
	appState.cleanup = append(appState.cleanup, redisClient.Close)

	// initialize HTTP server
	app := fiber.New()

	app.Use(requestid.New())
	app.Use(logger.New())
	app.Use(recover.New())

	// register application routes
	health.RegisterRoutes(app,
		health.Params{
			Database:    database.Client,
			Redis:       redisClient,
			MinIO:       minioClient,
			MinIOBucket: cfg.MinIO.Bucket,
		})

	categories.RegisterRoutes(
		app,
		database.Client,
		redisClient,
		log,
		validate,
	)

	appState.Server = app

	log.Info("Fiber application initialized")

	return appState, nil
}

func (a *Application) Close() error {
	var errs []error

	// close dependencies in reverse initialization order
	for i := len(a.cleanup) - 1; i >= 0; i-- {
		if err := a.cleanup[i](); err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}
