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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/igor/gophkeeper/internal/config"
	appcrypto "github.com/igor/gophkeeper/internal/crypto"
	"github.com/igor/gophkeeper/internal/middleware"
	"github.com/igor/gophkeeper/internal/modules/auth/domain"
	authhandlers "github.com/igor/gophkeeper/internal/modules/auth/handlers"
	"github.com/igor/gophkeeper/internal/modules/auth/password"
	authpostgres "github.com/igor/gophkeeper/internal/modules/auth/repositories/postgres"
	"github.com/igor/gophkeeper/internal/modules/auth/token"
	"github.com/igor/gophkeeper/internal/modules/auth/usecases"
	infrahandlers "github.com/igor/gophkeeper/internal/modules/infra/handlers"
	secretsdomain "github.com/igor/gophkeeper/internal/modules/secrets/domain"
	secretshandlers "github.com/igor/gophkeeper/internal/modules/secrets/handlers"
	secretspostgres "github.com/igor/gophkeeper/internal/modules/secrets/repositories/postgres"
	secretsusecases "github.com/igor/gophkeeper/internal/modules/secrets/usecases"
	"github.com/igor/gophkeeper/internal/storage/filesystem"
	authapi "github.com/igor/gophkeeper/pkg/api/generated/auth"
	infraapi "github.com/igor/gophkeeper/pkg/api/generated/infra"
	secretsapi "github.com/igor/gophkeeper/pkg/api/generated/secrets"
)

// UserRepository сохраняет и загружает пользователей.
type UserRepository interface {
	Save(ctx context.Context, user domain.User) error
	Load(ctx context.Context, login string) (domain.User, error)
}

// SecretsRepository сохраняет и загружает JSON-секреты.
type SecretsRepository interface {
	Save(ctx context.Context, secret secretsdomain.Secret) error
	Load(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (secretsdomain.Secret, error)
	List(ctx context.Context, userID uuid.UUID) ([]secretsdomain.Secret, error)
	Update(ctx context.Context, secret secretsdomain.Secret) error
	Delete(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) error
	SaveBlob(ctx context.Context, blob secretsdomain.Blob) error
	LoadBlobBySecret(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (secretsdomain.Blob, error)
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
	secretsRepository, err := secretspostgres.NewRepository(pool)
	if err != nil {
		return fmt.Errorf("create secrets repository: %w", err)
	}

	router, err := BuildRouter(logger, cfg, userRepository, secretsRepository)
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
func BuildRouter(
	logger zerolog.Logger,
	cfg config.Config,
	userRepository UserRepository,
	secretsRepository SecretsRepository,
) (*chi.Mux, error) {
	if userRepository == nil {
		return nil, fmt.Errorf("%w: userRepository", usecases.ErrEmptyDependency)
	}
	if secretsRepository == nil {
		return nil, fmt.Errorf("%w: secretsRepository", secretsusecases.ErrEmptyDependency)
	}
	router := chi.NewRouter()
	infraapi.HandlerFromMux(infraapi.NewStrictHandler(infrahandlers.New(), nil), router)

	passwordHasher := password.NewBcryptHasher()
	tokenIssuer := token.NewJWTIssuer(cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTL)
	encryptor, err := appcrypto.NewAESGCMEncryptor(cfg.Crypto.MasterKey)
	if err != nil {
		return nil, fmt.Errorf("create encryptor: %w", err)
	}
	blobStorage, err := filesystem.NewBlobStorage(cfg.Storage.BlobPath)
	if err != nil {
		return nil, fmt.Errorf("create blob storage: %w", err)
	}
	validate := validator.New(validator.WithRequiredStructEnabled())

	registerUseCase, err := usecases.NewRegisterUseCase(userRepository, passwordHasher)
	if err != nil {
		return nil, fmt.Errorf("create register use case: %w", err)
	}
	loginUseCase, err := usecases.NewLoginUseCase(userRepository, passwordHasher, tokenIssuer)
	if err != nil {
		return nil, fmt.Errorf("create login use case: %w", err)
	}

	authHandler, err := authhandlers.New(logger, registerUseCase, loginUseCase, validate)
	if err != nil {
		return nil, fmt.Errorf("create auth handler: %w", err)
	}
	authapi.HandlerFromMux(authapi.NewStrictHandler(authHandler, nil), router)

	createSecretUseCase, err := secretsusecases.NewCreateUseCase(secretsRepository, encryptor)
	if err != nil {
		return nil, fmt.Errorf("create secret create use case: %w", err)
	}
	listSecretUseCase, err := secretsusecases.NewListUseCase(secretsRepository, encryptor)
	if err != nil {
		return nil, fmt.Errorf("create secret list use case: %w", err)
	}
	getSecretUseCase, err := secretsusecases.NewGetUseCase(secretsRepository, encryptor)
	if err != nil {
		return nil, fmt.Errorf("create secret get use case: %w", err)
	}
	updateSecretUseCase, err := secretsusecases.NewUpdateUseCase(secretsRepository, encryptor)
	if err != nil {
		return nil, fmt.Errorf("create secret update use case: %w", err)
	}
	deleteSecretUseCase, err := secretsusecases.NewDeleteUseCase(secretsRepository)
	if err != nil {
		return nil, fmt.Errorf("create secret delete use case: %w", err)
	}
	createBlobUseCase, err := secretsusecases.NewCreateBlobUseCase(secretsRepository, encryptor, blobStorage)
	if err != nil {
		return nil, fmt.Errorf("create blob secret use case: %w", err)
	}
	getBlobContentUseCase, err := secretsusecases.NewGetBlobContentUseCase(secretsRepository, encryptor, blobStorage)
	if err != nil {
		return nil, fmt.Errorf("create blob content use case: %w", err)
	}

	secretsHandler, err := secretshandlers.New(
		logger,
		createSecretUseCase,
		listSecretUseCase,
		getSecretUseCase,
		updateSecretUseCase,
		deleteSecretUseCase,
		createBlobUseCase,
		getBlobContentUseCase,
	)
	if err != nil {
		return nil, fmt.Errorf("create secrets handler: %w", err)
	}

	router.Group(func(protected chi.Router) {
		protected.Use(middleware.Auth(tokenIssuer))
		secretsapi.HandlerFromMux(secretsapi.NewStrictHandler(secretsHandler, nil), protected)
	})

	return router, nil
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
