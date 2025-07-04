package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"crypto/rand"

	"github.com/go-resty/resty/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"auth-service/internal/models"
	"auth-service/internal/repository"
	"auth-service/pkg/er"
)

type Config struct {
	JwtSecret  string        `env:"JWT_SECRET,required"`
	AccessTTL  time.Duration `env:"ACCESS_TTL,required"`
	RefreshTTL time.Duration `env:"REFRESH_TTL,required"`
	WebhookURL string        `env:"WEBHOOK_URL,required"`
	UserAgent  string        `env:"USER_AGENT"`
}

type Service struct {
	repo       repository.Repository
	jwtSecret  []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	client     *resty.Client
}

func NewService(repo repository.Repository, cfg Config) *Service {
	client := resty.New()
	client.SetBaseURL(cfg.WebhookURL)
	if cfg.UserAgent != "" {
		client.SetHeader("User-Agent", cfg.UserAgent)
	}
	s := &Service{
		repo:       repo,
		jwtSecret:  []byte(cfg.JwtSecret),
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
		client:     client,
	}
	return s
}

// GenerateTokens генерирует пару access и refresh токенов для пользователя
func (s *Service) GenerateTokens(ctx context.Context, userID uuid.UUID, userAgent, ip string) (string, string, error) {
	_, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, er.ErrNotFound) {
			return "", "", er.ErrNotFound
		}
		return "", "", fmt.Errorf("failed to get user by id %s: %w", userID, err)
	}

	expiresAt := time.Now().Add(s.accessTTL)
	accessToken, err := s.generateAccessToken(userID, expiresAt)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshTokenRaw, err := generateRandomBase64(32)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate random refresh token: %w", err)
	}
	refreshTokenHash, err := bcrypt.GenerateFromPassword([]byte(refreshTokenRaw), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash refresh token: %w", err)
	}

	rt := &models.RefreshToken{
		UserID:    userID,
		TokenHash: string(refreshTokenHash),
		UserAgent: userAgent,
		IP:        ip,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(s.refreshTTL),
		IsValid:   true,
	}
	if err := s.repo.CreateRefreshToken(ctx, rt); err != nil {
		return "", "", fmt.Errorf("failed to create refresh token: %w", err)
	}

	return accessToken, refreshTokenRaw, nil
}

// RefreshTokens обновляет пару токенов
func (s *Service) RefreshTokens(ctx context.Context, userID uuid.UUID, refreshTokenRaw, userAgent, ip string) (string, string, error) {
	refreshTokens, err := s.repo.GetValidUserRefreshTokens(ctx, userID)
	if err != nil {
		if errors.Is(err, er.ErrNotFound) {
			return "", "", er.ErrNotFound
		}
		return "", "", fmt.Errorf("failed to get valid refresh tokens for user %s: %w", userID, err)
	}
	var refreshToken *models.RefreshToken
	for _, t := range refreshTokens {
		if bcrypt.CompareHashAndPassword([]byte(t.TokenHash), []byte(refreshTokenRaw)) == nil {
			refreshToken = t
			break
		}
	}
	if refreshToken == nil {
		return "", "", er.ErrInvalidToken
	}
	if refreshToken.UserAgent != userAgent {
		_ = s.repo.InvalidateAllUserTokens(ctx, refreshToken.UserID)
		return "", "", er.ErrUserAgentMismatch
	}
	if refreshToken.IP != ip {
		err := s.Webhook(ctx, userID, ip)
		if err != nil {
			zap.S().Errorf("cannot send webhook: %s", err)
			return "", "", fmt.Errorf("failed to send webhook: %w", err)
		}
	}

	_ = s.repo.InvalidateRefreshToken(ctx, refreshToken.TokenHash)

	accessToken, refreshTokenRaw, err := s.GenerateTokens(ctx, refreshToken.UserID, userAgent, ip)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate new tokens: %w", err)
	}
	return accessToken, refreshTokenRaw, nil
}

// GetCurrentUserID возвращает userID по access токену
func (s *Service) GetCurrentUserID(accessToken string) (uuid.UUID, error) {
	claims, err := s.parseAccessToken(accessToken)
	if err != nil {
		if errors.Is(err, er.ErrInvalidToken) {
			return uuid.Nil, er.ErrInvalidToken
		}
		return uuid.Nil, fmt.Errorf("failed to parse access token: %w", err)
	}
	return claims.UserID, nil
}

// Logout деавторизует пользователя (инвалидирует все refresh токены)
func (s *Service) Logout(ctx context.Context, accessToken string) error {
	claims, err := s.parseAccessToken(accessToken)
	if err != nil {
		if errors.Is(err, er.ErrInvalidToken) {
			return er.ErrInvalidToken
		}
		return fmt.Errorf("failed to parse access token: %w", err)
	}
	if err := s.repo.InvalidateAllUserTokens(ctx, claims.UserID); err != nil {
		return fmt.Errorf("failed to invalidate all user tokens: %w", err)
	}
	return nil
}

func (s *Service) Webhook(ctx context.Context, userID uuid.UUID, ip string) error {
	req := WebhookRequest{
		NewIP:  ip,
		UserID: userID,
		Ts:     time.Now().Unix(),
	}

	resp, err := s.client.R().SetBody(req).Post("")
	if err != nil {
		return fmt.Errorf("failed to send webhook: %w", err)
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return fmt.Errorf("webhook returned non-success status: %d, body: %s", resp.StatusCode(), resp.String())
	}
	return nil
}

func (s *Service) generateAccessToken(userID uuid.UUID, expiresAt time.Time) (string, error) {
	claims := models.AccessTokenClaims{
		UserID:    userID,
		ExpiresAt: expiresAt.Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}
	return signed, nil
}

func (s *Service) parseAccessToken(tokenStr string) (*models.AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &models.AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse access token: %w", err)
	}
	claims, ok := token.Claims.(*models.AccessTokenClaims)
	if !ok || !token.Valid {
		return nil, er.ErrInvalidToken
	}
	return claims, nil
}

func generateRandomBase64(n int) (string, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
