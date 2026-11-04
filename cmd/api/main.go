package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/adamandp/dam-creamy-cream-go/internal/app"
	"github.com/adamandp/dam-creamy-cream-go/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		log.Error("configuration", "err", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	runtime, err := app.Bootstrap(ctx, cfg, log)
	if err != nil {
		log.Error("bootstrap", "err", err)
		os.Exit(1)
	}
	defer func() {
		if err := runtime.Close(); err != nil {
			log.Error("shutdown dependencies", "err", err)
		}
	}()
	stopped := make(chan struct{})
	go func() {
		<-ctx.Done()
		if err := runtime.HTTP.Shutdown(); err != nil {
			log.Error("HTTP shutdown", "err", err)
		}
		close(stopped)
	}()
	log.Info("listening", "port", cfg.Port)
	if err := runtime.HTTP.Listen(":" + cfg.Port); err != nil {
		log.Error("HTTP server failed", "err", err)
		stop()
		<-stopped
		os.Exit(1)
	}
}
