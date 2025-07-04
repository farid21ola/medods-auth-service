package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"auth-service/internal/models"
	"auth-service/pkg/er"
)

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(connString string) (*Postgres, error) {
	pool, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

// GetUserByID получает пользователя по его UUID
func (p *Postgres) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `SELECT id, created_at, updated_at FROM users WHERE id = $1`
	var user models.User
	err := p.pool.QueryRow(ctx, query, id).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, er.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by id %s: %w", id, err)
	}
	return &user, nil
}

// CreateRefreshToken сохраняет refresh токен
func (p *Postgres) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	query := `INSERT INTO refresh_tokens (user_id, token_hash, user_agent, ip, issued_at, expires_at, is_valid) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := p.pool.Exec(ctx, query, token.UserID, token.TokenHash, token.UserAgent, token.IP, token.IssuedAt, token.ExpiresAt, token.IsValid)
	if err != nil {
		return fmt.Errorf("failed to create refresh token for user %s: %w", token.UserID, err)
	}
	return nil
}

// GetRefreshToken получает refresh токен по хешу
func (p *Postgres) GetRefreshToken(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	query := `SELECT id, user_id, token_hash, user_agent, ip, issued_at, expires_at, is_valid FROM refresh_tokens WHERE token_hash = $1`
	var token models.RefreshToken
	err := p.pool.QueryRow(ctx, query, tokenHash).Scan(&token.ID, &token.UserID, &token.TokenHash, &token.UserAgent, &token.IP, &token.IssuedAt, &token.ExpiresAt, &token.IsValid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, er.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get refresh token by hash: %w", err)
	}
	return &token, nil
}

// InvalidateRefreshToken делает refresh токен невалидным
func (p *Postgres) InvalidateRefreshToken(ctx context.Context, tokenHash string) error {
	query := `UPDATE refresh_tokens SET is_valid = false WHERE token_hash = $1`
	cmd, err := p.pool.Exec(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to invalidate refresh token: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return er.ErrNotFound
	}
	return nil
}

// InvalidateAllUserTokens делает все refresh токены пользователя невалидными
func (p *Postgres) InvalidateAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE refresh_tokens SET is_valid = false WHERE user_id = $1`
	_, err := p.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to invalidate all tokens for user %s: %w", userID, err)
	}
	return nil
}

// GetUserRefreshTokens получает все refresh токены пользователя
func (p *Postgres) GetUserRefreshTokens(ctx context.Context, userID uuid.UUID) ([]*models.RefreshToken, error) {
	query := `SELECT id, user_id, token_hash, user_agent, ip, issued_at, expires_at, is_valid FROM refresh_tokens WHERE user_id = $1`
	rows, err := p.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh tokens for user %s: %w", userID, err)
	}
	defer rows.Close()

	var tokens []*models.RefreshToken
	for rows.Next() {
		var token models.RefreshToken
		if err := rows.Scan(&token.ID, &token.UserID, &token.TokenHash, &token.UserAgent, &token.IP, &token.IssuedAt, &token.ExpiresAt, &token.IsValid); err != nil {
			return nil, fmt.Errorf("failed to scan refresh token for user %s: %w", userID, err)
		}
		tokens = append(tokens, &token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan refresh tokens for user %s: %w", userID, err)
	}
	return tokens, nil
}

// GetValidUserRefreshTokens получает только валидные refresh токены пользователя
func (p *Postgres) GetValidUserRefreshTokens(ctx context.Context, userID uuid.UUID) ([]*models.RefreshToken, error) {
	query := `SELECT id, user_id, token_hash, user_agent, ip, issued_at, expires_at, is_valid FROM refresh_tokens WHERE user_id = $1 AND is_valid = true AND expires_at > NOW()`
	rows, err := p.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid refresh tokens for user %s: %w", userID, err)
	}
	defer rows.Close()

	var tokens []*models.RefreshToken
	for rows.Next() {
		var token models.RefreshToken
		if err := rows.Scan(&token.ID, &token.UserID, &token.TokenHash, &token.UserAgent, &token.IP, &token.IssuedAt, &token.ExpiresAt, &token.IsValid); err != nil {
			return nil, fmt.Errorf("failed to scan valid refresh token for user %s: %w", userID, err)
		}
		tokens = append(tokens, &token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan valid refresh tokens for user %s: %w", userID, err)
	}
	return tokens, nil
}
