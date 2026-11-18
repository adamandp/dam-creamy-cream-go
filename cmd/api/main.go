package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"dam-creamy-cream-go/internal/config"
	"dam-creamy-cream-go/internal/infrastructure"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

func main() {
	// create one application context for startup and shutdown
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// initialize shared validation
	validate := validator.New(
		validator.WithRequiredStructEnabled(),
	)

	// load env
	cfg, err := config.Load(validate)
	if err != nil {
		fmt.Fprintln(os.Stderr, "configuration error:", err)
		os.Exit(1)
	}

	// initialize logging
	log := infrastructure.New(cfg.AppEnv)

	// bootstrap shared application dependencies
	application, err := infrastructure.Bootstrap(
		ctx,
		cfg,
		log,
		validate,
	)
	if err != nil {
		log.WithError(err).Error("Application bootstrap failed")
		os.Exit(1)
	}
	defer func() {
		if err := application.Close(); err != nil {
			log.WithError(err).Error("Application cleanup failed")
		}
	}()

	// start the server
	if err := application.Server.Listen(
		":"+strconv.Itoa(cfg.Port),
		fiber.ListenConfig{GracefulContext: ctx},
	); err != nil {
		log.WithError(err).Error("HTTP server stopped")
	}
}
