package database

import (
	"Kite/internal/config"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	MaxConnLifetime = time.Hour
	MaxConnIdleTime = 30 * time.Minute
)

type DB interface {
	Pool() *pgxpool.Pool
	Close()
}

type Config struct {
	Host     string
	Port     uint16
	User     string
	Password string
	Name     string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

type postgres struct {
	pool *pgxpool.Pool
}

func (p *postgres) Pool() *pgxpool.Pool {
	return p.pool
}

func (p *postgres) Close() {
	p.pool.Close()
}

func New(ctx context.Context, cfg *config.Postgres) (DB, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	// строка подключения
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.SSLMode,
	)

	// конфиг настройки pgxpool
	parseConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	parseConfig.MaxConns = cfg.MaxConns
	parseConfig.MinConns = cfg.MinConns
	parseConfig.MaxConnLifetime = MaxConnLifetime
	parseConfig.MaxConnIdleTime = MaxConnIdleTime

	//возвращает pool из конфига
	pool, err := pgxpool.NewWithConfig(context.Background(), parseConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	m, err := migrate.New("file://migrations", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration instance: %w", err)
	}

	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil || dbErr != nil {
			err = fmt.Errorf("failed to close migration instance: source error: %w, database error: %w", srcErr, dbErr)
		}
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &postgres{pool: pool}, nil
}
