package repository

import (
	"auth-service/internal/models"
	"auth-service/internal/repository/postgres"
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Config struct {
	DBHost     string `env:"DB_HOST,required"`
	DBPort     string `env:"DB_PORT,required"`
	DBUser     string `env:"DB_USER,required"`
	DBPassword string `env:"DB_PASSWORD,required"`
	DBName     string `env:"DB_NAME,required"`
}

func NewRepository(cfg Config) (Repository, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)
	return postgres.NewPostgres(connStr)
}

type Repository interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)

	CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	InvalidateRefreshToken(ctx context.Context, tokenHash string) error
	InvalidateAllUserTokens(ctx context.Context, userID uuid.UUID) error

	GetUserRefreshTokens(ctx context.Context, userID uuid.UUID) ([]*models.RefreshToken, error)
	GetValidUserRefreshTokens(ctx context.Context, userID uuid.UUID) ([]*models.RefreshToken, error)
}
