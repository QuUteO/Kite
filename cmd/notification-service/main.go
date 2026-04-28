package main

import (
	"Kite/internal/config"
	"Kite/internal/notification"
	_ "Kite/internal/notification"
	slogger "Kite/pkg/logger"
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.LoadConfig("./config/config.yml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger := slogger.SetupLogger(cfg.Env)

	service := notification.NewService(cfg, logger)

	if err := service.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("service failed", "error", err)
		os.Exit(1)
	}

	logger.Info("notification-service service stopped")
}
