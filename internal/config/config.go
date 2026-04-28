// internal/config/config.go - альтернативный вариант с YAML тегами
package config

import (
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env      string     `yaml:"ENV" default:"dev"`
	Postgres Postgres   `yaml:"POSTGRES"`
	HTTP     HTTPServer `yaml:"HTTP"`
	JWT      JWT        `yaml:"JWT"`
	RabbitMQ RabbitMQ   `yaml:"RABBITMQ"`
	SMTP     SMTP       `yaml:"SMTP"`
}

type Postgres struct {
	Host     string `yaml:"POSTGRES_HOST" default:"localhost"`
	Port     uint16 `yaml:"POSTGRES_PORT" default:"5432"`
	User     string `yaml:"POSTGRES_USER" default:"postgres"`
	Password string `yaml:"POSTGRES_PASSWORD" default:"postgres"`
	Database string `yaml:"POSTGRES_DB" default:"postgres"`
	SSLMode  string `yaml:"SSL_MODE" default:"disable"`
	MaxConns int32  `yaml:"MAX_CONNS" default:"10"`
	MinConns int32  `yaml:"MIN_CONNS" default:"5"`
}

type HTTPServer struct {
	Address      string        `yaml:"ADDRESS" default:"localhost:8080"`
	ReadTimeout  time.Duration `yaml:"READ_TIMEOUT" default:"10s"`
	WriteTimeout time.Duration `yaml:"WRITE_TIMEOUT" default:"10s"`
	IdleTimeout  time.Duration `yaml:"IDLE_TIMEOUT" default:"120s"`
}

type JWT struct {
	Secret     string        `yaml:"SECRET" env-required:"true"`
	AccessTTL  time.Duration `yaml:"ACCESS_TTL" default:"15m"`
	RefreshTTL time.Duration `yaml:"REFRESH_TTL" default:"24h"`
}

type RabbitMQ struct {
	URL                string `yaml:"URL" env-required:"true"`
	Exchange           string `yaml:"EXCHANGE" default:"kite.events"`
	RegisterQueue      string `yaml:"REGISTER_QUEUE" default:"auth.email.register"`
	RegisterRoutingKey string `yaml:"REGISTER_ROUTING_KEY" default:"auth.email.register"`
}

type SMTP struct {
	From     string `yaml:"FROM" env-required:"true"`
	Password string `yaml:"PASSWORD" env-required:"true"`
	Host     string `yaml:"HOST" env-required:"true"`
	Port     string `yaml:"PORT" default:"587"`
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
