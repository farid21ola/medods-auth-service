package main

import (
	"log"
	"net/http"

	"go.uber.org/zap"

	"auth-service/config"
	"auth-service/internal/httpserver"
	"auth-service/internal/httpserver/handler"
	"auth-service/internal/httpserver/handler/middleware/auth"
	"auth-service/internal/httpserver/handler/middleware/ip"
	"auth-service/internal/repository"
	"auth-service/internal/service"
	"auth-service/pkg/logger"
)

// @title           Medods Auth Service API
// @version         1.0
// @description     Сервис аутентификации пользователей для Medods.
// @host            localhost:8081
// @BasePath        /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal("failed to load config: ", err)
	}

	logger.SetupLogger(cfg.LoggerConfig)
	zap.S().Info("logger initialized")

	repo, err := repository.NewRepository(cfg.RepositoryConfig)
	if err != nil {
		zap.S().Fatalf("failed to initialize repository: %s", err)
	}
	zap.S().Info("repository initialized")

	svc := service.NewService(repo, cfg.ServiceConfig)
	zap.S().Info("service initialized")

	// ToDO: swagger описать и docker-compose, посмотреть как что с логированием у нас
	h := handler.NewHandler(svc)
	authMiddleware := auth.Middleware(svc.GetCurrentUserID)
	ipMiddleware := ip.Middleware

	server := httpserver.CreateServer(cfg.ServerConfig, h, authMiddleware, ipMiddleware)

	zap.S().Infof("starting server on %s", cfg.ServerConfig.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		zap.S().Fatalf("server failed: %v", err)
	}
}
