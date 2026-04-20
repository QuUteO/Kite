package main

import (
	"Kite/internal/app"
	"Kite/internal/config"
	"Kite/pkg/logger"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadConfig("./config/config.yml")
	if err != nil {
		log.Fatalf("error: %s", err)
	}

	logger := slogger.SetupLogger(cfg.Env)

	if err := app.Run(ctx, cfg, logger); err != nil {
		logger.Error("error while starting server: ", err)
		os.Exit(1)
	}
}
