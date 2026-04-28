package main

import (
	"Kite/internal/config"
	"Kite/internal/user/app"
	slogger "Kite/pkg/logger"
	"context"
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

	if err := app.Run(ctx, cfg, logger); err != nil {
		logger.Error("application failed", "error", err)
		os.Exit(1)
	}

	logger.Info("user service stopped")
}
