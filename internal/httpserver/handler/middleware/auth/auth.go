package auth

import (
	"auth-service/internal/httpserver/handler"
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TokenValidator func(token string) (uuid.UUID, error)

func Middleware(validate TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("Content-Type", "application/json")
				handler.WriteJSONResponse(w, http.StatusUnauthorized, handler.Response{
					Status: "error",
					Msg:    "missing access token",
				})
				return
			}
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				w.Header().Set("Content-Type", "application/json")
				handler.WriteJSONResponse(w, http.StatusUnauthorized, handler.Response{
					Status: "error",
					Msg:    "invalid authorization header",
				})
				return
			}
			token := parts[1]
			guid, err := validate(token)
			if err != nil || guid == uuid.Nil {
				zap.S().Infof("auth middleware: invalid access token: %v", err)
				w.Header().Set("Content-Type", "application/json")
				handler.WriteJSONResponse(w, http.StatusUnauthorized, handler.Response{
					Status: "error",
					Msg:    "invalid access token",
				})
				return
			}
			ctx := context.WithValue(r.Context(), handler.ContextKeyGUID, guid.String())
			ctx = context.WithValue(ctx, handler.ContextKeyAccessToken, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
