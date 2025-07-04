package config

import (
	"fmt"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"

	"auth-service/internal/httpserver"
	"auth-service/internal/repository"
	"auth-service/internal/service"
	"auth-service/pkg/logger"
)

type Config struct {
	LoggerConfig     logger.Config
	RepositoryConfig repository.Config
	ServiceConfig    service.Config
	ServerConfig     httpserver.Config
}

func NewConfig() (*Config, error) {
	godotenv.Load(".env")

	cfg := Config{}

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	return &cfg, nil
}
