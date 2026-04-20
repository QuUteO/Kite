package main

import (
	"Kite/internal/app/notification"
	"Kite/internal/config"
	slogger "Kite/pkg/logger"
	"Kite/pkg/rabbitmq"
	"context"
	"log"
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

	// ПРОВЕРКА: создаем временное соединение для проверки
	connManager := rabbitmq.NewConnectionManager(cfg.RabbitMQ.URL, logger)
	conn, err := connManager.GetConnection()
	if err != nil {
		logger.Error("Failed to connect: %v", err)
	}

	// Проверяем очередь
	_, err = conn.Channel.QueueDeclare(
		cfg.RabbitMQ.EmailRegisterQueue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		logger.Error("Failed to declare queue: %v", err)
	}

	// Проверяем binding
	err = conn.Channel.QueueBind(
		cfg.RabbitMQ.EmailRegisterQueue,
		cfg.RabbitMQ.EmailLoginRoutingKey,
		cfg.RabbitMQ.Exchange,
		false,
		nil,
	)
	if err != nil {
		logger.Error("Binding might not exist: %v", err)
	} else {
		logger.Error("Binding exists or created")
	}

	service := notification.NewNotification(cfg, logger)

	if err := service.Run(ctx); err != nil {
		logger.Error("Failed to run: %v", err)
	}

	select {}
}
