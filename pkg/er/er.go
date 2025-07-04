package er

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrInvalidToken      = errors.New("invalid token")
	ErrUserAgentMismatch = errors.New("user agent mismatch")
	ErrTokenExpired      = errors.New("token expired")
)
