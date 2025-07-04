package httpserver

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "auth-service/docs"
	"auth-service/internal/httpserver/handler"
)

type Config struct {
	Port        string        `env:"SERVER_PORT" envDefault:"8081"`
	Timeout     time.Duration `env:"TIMEOUT" envDefault:"10s"`
	IdleTimeout time.Duration `env:"IDLE_TIMEOUT" envDefault:"60s"`
}

func CreateServer(cfg Config, handler *handler.Handler, authMiddleware, ipMiddleware func(http.Handler) http.Handler) *http.Server {
	r := mux.NewRouter()

	r.Use(ipMiddleware)

	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/tokens/refresh", handler.RefreshTokens()).Methods(http.MethodPost)
	api.HandleFunc("/tokens/{guid}", handler.GenerateTokens()).Methods(http.MethodPost)

	protected := api.NewRoute().Subrouter()
	protected.Use(authMiddleware)
	protected.HandleFunc("/me", handler.GetMe()).Methods(http.MethodGet)
	protected.HandleFunc("/logout", handler.Logout()).Methods(http.MethodPost)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return server
}
