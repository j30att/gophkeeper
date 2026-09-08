// Package main содержит точку входа команды уборки blob-файлов.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/igor/gophkeeper/internal/config"
	secretspostgres "github.com/igor/gophkeeper/internal/modules/secrets/repositories/postgres"
	secretsusecases "github.com/igor/gophkeeper/internal/modules/secrets/usecases"
	"github.com/igor/gophkeeper/internal/storage/filesystem"
)

// main запускает ручную уборку файлов blob-секретов.
func main() {
	if err := run(context.Background()); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "cleanup failed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg := config.Load()
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	pool, err := pgxpool.New(ctx, cfg.Postgres.DSN)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer pool.Close()

	repository, err := secretspostgres.NewRepository(pool)
	if err != nil {
		return fmt.Errorf("create secrets repository: %w", err)
	}
	storage, err := filesystem.NewBlobStorage(cfg.Storage.BlobPath)
	if err != nil {
		return fmt.Errorf("create blob storage: %w", err)
	}
	useCase, err := secretsusecases.NewCleanupBlobsUseCase(repository, storage)
	if err != nil {
		return fmt.Errorf("create cleanup use case: %w", err)
	}

	output, err := useCase.Execute(ctx, secretsusecases.CleanupBlobsInput{Before: time.Now().UTC()})
	if err != nil {
		return err
	}
	logger.Info().Int("deleted", output.Deleted).Msg("blob cleanup finished")
	return nil
}
