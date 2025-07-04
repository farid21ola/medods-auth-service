package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"auth-service/internal/service"
	"auth-service/pkg/er"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// GenerateTokens
// @Summary      Генерация access и refresh токенов
// @Description  Генерирует пару токенов по guid пользователя
// @Tags         auth
// @Param        guid path string true "GUID пользователя"
// @Success      200 {object} Response
// @Failure      400 {object} Response "guid не передан или неверный формат"
// @Failure      404 {object} Response "Пользователь не найден"
// @Failure      500 {object} Response "Внутренняя ошибка сервера"
// @Router       /tokens/{guid} [post]
func (h *Handler) GenerateTokens() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zap.S().Infof("GenerateTokens handler start")
		vars := mux.Vars(r)
		guidVal := vars["guid"]
		if guidVal == "" {
			zap.S().Warnf("received request without guid")
			WriteJSONResponse(w, http.StatusBadRequest, Response{
				Status: "error",
				Msg:    "guid is required",
			})
			zap.S().Warnf("GenerateTokens handler error: guid is required")
			return
		}
		guid, err := uuid.Parse(guidVal)
		if err != nil {
			zap.S().Warnf("received request with invalid guid format: %s", guidVal)
			WriteJSONResponse(w, http.StatusBadRequest, Response{
				Status: "error",
				Msg:    "invalid guid format",
			})
			zap.S().Warnf("GenerateTokens handler error: invalid guid format")
			return
		}

		userAgent := r.UserAgent()
		ipVal := r.Context().Value(ContextKeyIP)
		ip, ok := ipVal.(string)
		if !ok {
			zap.S().Errorf("ip not found in context for guid: %s", guid.String())
			WriteJSONResponse(w, http.StatusInternalServerError, Response{
				Status: "error",
				Msg:    "internal server error",
			})
			zap.S().Errorf("GenerateTokens handler error: ip not found in context")
			return
		}

		at, rt, err := h.svc.GenerateTokens(r.Context(), guid, userAgent, ip)
		if err != nil {
			if errors.Is(err, er.ErrNotFound) {
				zap.S().Infof("user not found: %v", err)
				WriteJSONResponse(w, http.StatusNotFound, Response{
					Status: "error",
					Msg:    "user not found",
					Data:   nil,
				})
				zap.S().Warnf("GenerateTokens handler error: user not found")
				return
			}
			zap.S().Errorf("failed to generate tokens: %v", err)
			WriteJSONResponse(w, http.StatusInternalServerError, Response{
				Status: "error",
				Msg:    "internal server error",
				Data:   nil,
			})
			zap.S().Errorf("GenerateTokens handler error: failed to generate tokens")
			return
		}

		data := TokenPair{
			AccessToken:  at,
			RefreshToken: rt,
		}

		WriteJSONResponse(w, http.StatusOK, Response{
			Status: "ok",
			Msg:    "",
			Data:   data,
		})
		zap.S().Infof("GenerateTokens handler success")
	}
}

// RefreshTokens
// @Summary      Обновление access и refresh токенов
// @Description  Обновляет пару токенов по refresh токену
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body RefreshTokensRequest true "Тело запроса"
// @Success      200 {object} Response
// @Failure      400 {object} Response "Некорректное тело запроса"
// @Failure      401 {object} Response "Неверный access или refresh токен"
// @Failure      404 {object} Response "Пользователь не найден"
// @Failure      500 {object} Response "Внутренняя ошибка сервера"
// @Router       /tokens/refresh [post]
func (h *Handler) RefreshTokens() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zap.S().Infof("RefreshTokens handler start")
		var req RefreshTokensRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			zap.S().Warnf("invalid refresh tokens request: %v", err)
			WriteJSONResponse(w, http.StatusBadRequest, Response{
				Status: "error",
				Msg:    "invalid request body",
			})
			zap.S().Warnf("RefreshTokens handler error: invalid request body")
			return
		}
		userAgent := r.UserAgent()
		ipVal := r.Context().Value(ContextKeyIP)
		ip, _ := ipVal.(string)

		userID, err := h.svc.GetCurrentUserID(req.AccessToken)
		if err != nil {
			zap.S().Errorf("invalid access token in refresh: %v", err)
			WriteJSONResponse(w, http.StatusUnauthorized, Response{
				Status: "error",
				Msg:    "invalid access token",
			})
			zap.S().Errorf("RefreshTokens handler error: invalid access token")
			return
		}

		at, rt, err := h.svc.RefreshTokens(r.Context(), userID, req.RefreshToken, userAgent, ip)
		if err != nil {
			if errors.Is(err, er.ErrNotFound) {
				WriteJSONResponse(w, http.StatusNotFound, Response{
					Status: "error",
					Msg:    "user not found",
				})
				zap.S().Warnf("RefreshTokens handler error: user not found")
				return
			}
			if errors.Is(err, er.ErrInvalidToken) {
				WriteJSONResponse(w, http.StatusUnauthorized, Response{
					Status: "error",
					Msg:    "invalid refresh token",
				})
				zap.S().Warnf("RefreshTokens handler error: invalid refresh token")
				return
			}
			if errors.Is(err, er.ErrUserAgentMismatch) {
				WriteJSONResponse(w, http.StatusUnauthorized, Response{
					Status: "error",
					Msg:    "user-agent mismatch, user deauthorized",
				})
				zap.S().Warnf("RefreshTokens handler error: user-agent mismatch")
				return
			}
			zap.S().Errorf("failed to refresh tokens: %v", err)
			WriteJSONResponse(w, http.StatusInternalServerError, Response{
				Status: "error",
				Msg:    "internal server error",
			})
			zap.S().Errorf("RefreshTokens handler error: failed to refresh tokens")
			return
		}

		WriteJSONResponse(w, http.StatusOK, Response{
			Status: "ok",
			Data:   TokenPair{AccessToken: at, RefreshToken: rt},
		})
		zap.S().Infof("RefreshTokens handler success")
	}
}

// GetMe
// @Summary      Получить информацию о себе
// @Description  Возвращает GUID текущего пользователя по access токену
// @Tags         auth
// @Produce      json
// @Success      200 {object} Response
// @Failure      401 {object} Response "Отсутствует или неверный access токен"
// @Router       /me [get]
// @Security     BearerAuth
func (h *Handler) GetMe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zap.S().Infof("GetMe handler start")
		accessTokenVal := r.Context().Value(ContextKeyAccessToken)
		accessToken, ok := accessTokenVal.(string)
		if !ok || accessToken == "" {
			zap.S().Warnf("missing or invalid access token in context")
			WriteJSONResponse(w, http.StatusUnauthorized, Response{
				Status: "error",
				Msg:    "missing or invalid access token",
			})
			zap.S().Warnf("GetMe handler error: missing or invalid access token")
			return
		}
		userID, err := h.svc.GetCurrentUserID(accessToken)
		if err != nil {
			zap.S().Warnf("invalid access token: %v", err)
			WriteJSONResponse(w, http.StatusUnauthorized, Response{
				Status: "error",
				Msg:    "invalid access token",
			})
			zap.S().Warnf("GetMe handler error: invalid access token")
			return
		}
		WriteJSONResponse(w, http.StatusOK, Response{
			Status: "ok",
			Data:   MeResponse{GUID: userID.String()},
		})
		zap.S().Infof("GetMe handler success")
	}
}

// Logout
// @Summary      Выход пользователя
// @Description  Инвалидирует access токен пользователя
// @Tags         auth
// @Produce      json
// @Success      200 {object} Response "Успешный выход"
// @Failure      401 {object} Response "Отсутствует или неверный access токен"
// @Failure      500 {object} Response "Внутренняя ошибка сервера"
// @Router       /logout [post]
// @Security     BearerAuth
func (h *Handler) Logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		zap.S().Infof("Logout handler start")
		accessTokenVal := r.Context().Value(ContextKeyAccessToken)
		accessToken, ok := accessTokenVal.(string)
		if !ok || accessToken == "" {
			zap.S().Warnf("missing or invalid access token in context (logout)")
			WriteJSONResponse(w, http.StatusUnauthorized, Response{
				Status: "error",
				Msg:    "missing or invalid access token",
			})
			zap.S().Warnf("Logout handler error: missing or invalid access token")
			return
		}
		if err := h.svc.Logout(r.Context(), accessToken); err != nil {
			if errors.Is(err, er.ErrInvalidToken) {
				zap.S().Warnf("invalid access token on logout: %v", err)
				WriteJSONResponse(w, http.StatusUnauthorized, Response{
					Status: "error",
					Msg:    "invalid access token",
				})
				zap.S().Warnf("Logout handler error: invalid access token")
				return
			}
			zap.S().Errorf("failed to logout: %v", err)
			WriteJSONResponse(w, http.StatusInternalServerError, Response{
				Status: "error",
				Msg:    "internal server error",
			})
			zap.S().Errorf("Logout handler error: failed to logout")
			return
		}
		WriteJSONResponse(w, http.StatusOK, Response{
			Status: "ok",
			Msg:    "logout successful",
		})
		zap.S().Infof("Logout handler success")
	}
}
