package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// User представляет пользователя системы
type User struct {
	ID        uuid.UUID `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// RefreshToken представляет refresh токен пользователя
type RefreshToken struct {
	ID        int       `db:"id" json:"id"`
	UserID    uuid.UUID `db:"user_id" json:"user_id"`
	TokenHash string    `db:"token_hash" json:"-"`
	UserAgent string    `db:"user_agent" json:"user_agent"`
	IP        string    `db:"ip" json:"ip"`
	IssuedAt  time.Time `db:"issued_at" json:"issued_at"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	IsValid   bool      `db:"is_valid" json:"is_valid"`
}

// AccessTokenClaims используется для генерации и проверки JWT access токена
// Не хранится в базе, только для работы с JWT
type AccessTokenClaims struct {
	UserID    uuid.UUID `json:"user_id"`
	ExpiresAt int64     `json:"exp"`
	jwt.RegisteredClaims
}
