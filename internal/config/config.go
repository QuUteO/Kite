package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Postgres Postgres   `yaml:"POSTGRES"`
	Env      string     `yaml:"ENV" default:"dev"`
	HTTP     HTTPServer `yaml:"HTTP"`
	JWT      JWT        `yaml:"JWT"`
	RabbitMQ RabbitMQ   `yaml:"RABBITMQ"`
}

type Postgres struct {
	PostgresHost     string `yaml:"POSTGRES_HOST" default:"5432"`
	PostgresPort     uint16 `yaml:"POSTGRES_PORT" env-default:"5432"`
	PostgresUser     string `yaml:"POSTGRES_USER" env-default:"postgres"`
	PostgresPassword string `yaml:"POSTGRES_PASSWORD" env-default:"postgres"`
	PostgresDatabase string `yaml:"POSTGRES_DB" env-default:"postgres"`
	SSLMode          string `yaml:"SSL_MODE" default:"disable"`
	MaxConnections   int32  `yaml:"MAX_CONNS" default:"10"`
	MinConnections   int32  `yaml:"MIN_CONNS" default:"2"`
}

type HTTPServer struct {
	Address      string        `yaml:"ADDRESS" default:"localhost:8081"`
	ReadTimeout  time.Duration `yaml:"READ_TIMEOUT" default:"10s"`
	WriteTimeout time.Duration `yaml:"WRITE_TIMEOUT" default:"10s"`
	IdleTimeout  time.Duration `yaml:"IDLE_TIMEOUT" default:"10s"`
}

type JWT struct {
	Secret     string        `yaml:"SECRET" env-required:"true"`
	AccessTTL  time.Duration `yaml:"ACCESS_TTL" env-required:"true"`
	RefreshTTL time.Duration `yaml:"REFRESH_TTL" env-required:"true"`
}

type RabbitMQ struct {
	URL                  string `yaml:"URL"`
	Exchange             string `yaml:"EXCHANGE"`
	EmailLogin           string `yaml:"EMAIL_LOGIN_QUEUE"`
	EmailLoginRoutingKey string `yaml:"EMAIL_LOGIN_ROUTING_KEY"`
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config
	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
