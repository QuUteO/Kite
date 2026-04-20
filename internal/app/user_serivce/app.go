package user_serivce

import (
	"Kite/internal/config"
	"Kite/internal/user/handler"
	"Kite/internal/user/repository"
	"Kite/internal/user/service"
	"Kite/pkg/database"
	"Kite/pkg/rabbitmq"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Run(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	db, err := initDatabase(ctx, cfg)
	if err != nil {
		return fmt.Errorf("init database failed: %w", err)
	}
	defer db.Close()

	rmqManager, err := initRabbitMQ(ctx, cfg, logger)
	if err != nil {
		return fmt.Errorf("init rabbitmq failed: %w", err)
	}

	router := newRouter(cfg, logger, db, rmqManager)
	server := newHTTPServer(cfg, router)

	return runHTTPServer(ctx, server, logger)
}

func initDatabase(ctx context.Context, cfg *config.Config) (database.DB, error) {
	db, err := database.New(ctx, &cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("initialize database: %w", err)
	}

	if err := db.Pool().Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

func newRouter(cfg *config.Config, logger *slog.Logger, db database.DB, rmqManager rabbitmq.ConnectionManager) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(30 * time.Second))

	producer := rabbitmq.NewProducer(rmqManager, logger)

	r.Mount("/user", newUserHandler(cfg, logger, db, producer).Routes())

	return r
}

func newUserHandler(cfg *config.Config, logger *slog.Logger, db database.DB, producer rabbitmq.Publisher) *handler.Handler {
	userRepo := repository.New(db.Pool(), logger)
	userService := service.New(userRepo, logger, cfg.JWT.Secret, producer, cfg)

	return handler.New(userService, logger, cfg.JWT.Secret)
}

func newHTTPServer(cfg *config.Config, router http.Handler) *http.Server {
	return &http.Server{
		Addr:         cfg.HTTP.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}
}

func runHTTPServer(ctx context.Context, server *http.Server, logger *slog.Logger) error {
	serverErrCh := make(chan error, 1)

	go func() {
		logger.Info("server started", "addr", server.Addr)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}

		serverErrCh <- nil
	}()

	select {
	case err := <-serverErrCh:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

func initRabbitMQ(ctx context.Context, cfg *config.Config, logger *slog.Logger) (rabbitmq.ConnectionManager, error) {
	rmqManager := rabbitmq.NewConnectionManager(cfg.RabbitMQ.URL, logger)

	logger.Info("RabbitMQ manager initialized", "url", cfg.RabbitMQ.URL)

	conn, err := rmqManager.GetConnection()
	if err != nil {
		return nil, fmt.Errorf("get rabbitmq connection: %w", err)
	}

	if cfg.RabbitMQ.Exchange != "" {
		err = conn.Channel.ExchangeDeclare(
			cfg.RabbitMQ.Exchange,
			"topic",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to declare exchange: %w", err)
		}
	}

	return rmqManager, nil
}
