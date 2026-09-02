// Package bootstrap собирает зависимости сервера и запускает HTTP-сервер.
package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/igor/gophkeeper/internal/config"
	"github.com/igor/gophkeeper/internal/modules/auth/domain"
	loginhandler "github.com/igor/gophkeeper/internal/modules/auth/handlers/login"
	registerhandler "github.com/igor/gophkeeper/internal/modules/auth/handlers/register"
	"github.com/igor/gophkeeper/internal/modules/auth/password"
	"github.com/igor/gophkeeper/internal/modules/auth/repositories/memory"
	authpostgres "github.com/igor/gophkeeper/internal/modules/auth/repositories/postgres"
	"github.com/igor/gophkeeper/internal/modules/auth/token"
	"github.com/igor/gophkeeper/internal/modules/auth/usecases"
)

// UserRepository сохраняет и загружает пользователей.
type UserRepository interface {
	Save(ctx context.Context, user domain.User) error
	Load(ctx context.Context, login string) (domain.User, error)
}

// StartServer собирает зависимости и запускает HTTP-сервер GophKeeper.
func StartServer(ctx context.Context, version string) error {
	cfg := config.Load()
	logger := zerolog.New(os.Stdout).With().Timestamp().Str("version", version).Logger()
	pool, err := pgxpool.New(ctx, cfg.Postgres.DSN)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	userRepository, err := authpostgres.NewUserRepository(pool)
	if err != nil {
		return fmt.Errorf("create user repository: %w", err)
	}

	router, err := BuildRouter(logger, cfg, userRepository)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.Server.Address,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info().Str("addr", cfg.Server.Address).Msg("starting GophKeeper server")
	return runHTTPServer(ctx, logger, server.ListenAndServe, server.Shutdown)
}

// BuildRouter создает HTTP-router и регистрирует handlers приложения.
func BuildRouter(logger zerolog.Logger, cfg config.Config, userRepository UserRepository) (*chi.Mux, error) {
	if userRepository == nil {
		return nil, fmt.Errorf("%w: userRepository", usecases.ErrEmptyDependency)
	}
	router := chi.NewRouter()
	router.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	passwordHasher := password.NewBcryptHasher()
	tokenIssuer := token.NewJWTIssuer(cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTL)
	validate := validator.New(validator.WithRequiredStructEnabled())

	registerUseCase, err := usecases.NewRegisterUseCase(userRepository, passwordHasher)
	if err != nil {
		return nil, fmt.Errorf("create register use case: %w", err)
	}
	loginUseCase, err := usecases.NewLoginUseCase(userRepository, passwordHasher, tokenIssuer)
	if err != nil {
		return nil, fmt.Errorf("create login use case: %w", err)
	}

	registerHandler, err := registerhandler.New(logger, registerUseCase, validate)
	if err != nil {
		return nil, fmt.Errorf("create register handler: %w", err)
	}
	loginHandler, err := loginhandler.New(logger, loginUseCase, validate)
	if err != nil {
		return nil, fmt.Errorf("create login handler: %w", err)
	}

	router.Post("/api/v1/auth/register", registerHandler.ServeHTTP)
	router.Post("/api/v1/auth/login", loginHandler.ServeHTTP)

	return router, nil
}

// BuildInMemoryRouter создает HTTP-router с in-memory repository для unit-тестов и ранней отладки.
func BuildInMemoryRouter(logger zerolog.Logger) (*chi.Mux, error) {
	return BuildRouter(logger, config.Load(), memory.NewUserRepository())
}

func runHTTPServer(
	ctx context.Context,
	logger zerolog.Logger,
	listenAndServe func() error,
	shutdown func(context.Context) error,
) error {
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- listenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown server: %w", err)
		}
		logger.Info().Msg("GophKeeper server stopped")
		return nil
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil
	}
}
