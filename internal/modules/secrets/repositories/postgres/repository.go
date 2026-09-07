// Package postgres содержит Postgres repositories secrets-модуля.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
	"github.com/igor/gophkeeper/internal/modules/secrets/usecases"
)

const (
	saveSecretQuery = `
INSERT INTO secrets (
	id, user_id, type, name, metadata, metadata_nonce, payload, payload_nonce, blob_id, version, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	loadSecretQuery = `
SELECT id, user_id, type, name, metadata, metadata_nonce, payload, payload_nonce, blob_id, version, created_at, updated_at, deleted_at
FROM secrets
WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL`

	listSecretsQuery = `
SELECT id, user_id, type, name, metadata, metadata_nonce, payload, payload_nonce, blob_id, version, created_at, updated_at, deleted_at
FROM secrets
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY updated_at DESC, id DESC`

	updateSecretQuery = `
UPDATE secrets
SET type = $3, name = $4, metadata = $5, metadata_nonce = $6, payload = $7, payload_nonce = $8,
	blob_id = $9, version = $10, updated_at = $11
WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL AND version = $12`

	deleteSecretQuery = `
WITH deleted_secret AS (
	UPDATE secrets
	SET deleted_at = now(), updated_at = now(), version = version + 1
	WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL
	RETURNING blob_id
),
deleted_blob AS (
	UPDATE blobs
	SET deleted_at = now()
	WHERE user_id = $1
		AND id IN (SELECT blob_id FROM deleted_secret WHERE blob_id IS NOT NULL)
		AND deleted_at IS NULL
	RETURNING id
)
SELECT count(*) FROM deleted_secret`

	saveBlobQuery = `
INSERT INTO blobs (
	id, user_id, original_name, storage_name, storage_path, content_type, size, checksum_sha256, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	loadBlobBySecretQuery = `
SELECT b.id, b.user_id, b.original_name, b.storage_name, b.storage_path, b.content_type, b.size,
	b.checksum_sha256, b.created_at, b.deleted_at, b.storage_deleted_at
FROM blobs b
JOIN secrets s ON s.blob_id = b.id
WHERE s.user_id = $1 AND s.id = $2 AND s.deleted_at IS NULL AND b.deleted_at IS NULL`

	markBlobDeletedQuery = `
UPDATE blobs
SET deleted_at = now()
WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL`

	listBlobsForCleanupQuery = `
SELECT id, user_id, original_name, storage_name, storage_path, content_type, size,
	checksum_sha256, created_at, deleted_at, storage_deleted_at
FROM blobs
WHERE deleted_at IS NOT NULL AND storage_deleted_at IS NULL AND deleted_at < $1
ORDER BY deleted_at ASC, id ASC
LIMIT $2`

	markBlobStorageDeletedQuery = `
UPDATE blobs
SET storage_deleted_at = now()
WHERE user_id = $1 AND id = $2 AND deleted_at IS NOT NULL AND storage_deleted_at IS NULL`
)

// Pool выполняет SQL-запросы к Postgres.
type Pool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Repository хранит JSON-секреты в Postgres.
type Repository struct {
	pool Pool
}

// NewRepository создает Repository.
func NewRepository(pool Pool) (*Repository, error) {
	if pool == nil {
		return nil, fmt.Errorf("%w: pool", usecases.ErrEmptyDependency)
	}
	return &Repository{pool: pool}, nil
}

// Save сохраняет JSON-секрет.
func (r *Repository) Save(ctx context.Context, secret domain.Secret) error {
	_, err := r.pool.Exec(
		ctx,
		saveSecretQuery,
		secret.ID,
		secret.UserID,
		secret.Type,
		secret.Name,
		secret.Metadata,
		secret.MetadataNonce,
		secret.Payload,
		secret.PayloadNonce,
		secret.BlobID,
		secret.Version,
		secret.CreatedAt,
		secret.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert secret: %w", err)
	}
	return nil
}

// Load загружает JSON-секрет по owner и id.
func (r *Repository) Load(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (domain.Secret, error) {
	secret, err := scanSecret(r.pool.QueryRow(ctx, loadSecretQuery, userID, secretID))
	if err != nil {
		return domain.Secret{}, err
	}
	return secret, nil
}

// List возвращает JSON-секреты пользователя.
func (r *Repository) List(ctx context.Context, userID uuid.UUID) ([]domain.Secret, error) {
	rows, err := r.pool.Query(ctx, listSecretsQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("select secrets: %w", err)
	}
	defer rows.Close()

	result := make([]domain.Secret, 0)
	for rows.Next() {
		secret, err := scanSecret(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, secret)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate secrets: %w", err)
	}
	return result, nil
}

// Update обновляет JSON-секрет по ожидаемой версии.
func (r *Repository) Update(ctx context.Context, secret domain.Secret, expectedVersion int) error {
	tag, err := r.pool.Exec(
		ctx,
		updateSecretQuery,
		secret.UserID,
		secret.ID,
		secret.Type,
		secret.Name,
		secret.Metadata,
		secret.MetadataNonce,
		secret.Payload,
		secret.PayloadNonce,
		secret.BlobID,
		secret.Version,
		secret.UpdatedAt,
		expectedVersion,
	)
	if err != nil {
		return fmt.Errorf("update secret: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return usecases.ErrSecretVersionConflict
	}
	return nil
}

// SaveBlob сохраняет метаданные blob-файла.
func (r *Repository) SaveBlob(ctx context.Context, blob domain.Blob) error {
	_, err := r.pool.Exec(
		ctx,
		saveBlobQuery,
		blob.ID,
		blob.UserID,
		blob.OriginalName,
		blob.StorageName,
		blob.StoragePath,
		blob.ContentType,
		blob.Size,
		blob.ChecksumSHA256,
		blob.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert blob: %w", err)
	}
	return nil
}

// LoadBlobBySecret загружает blob по owner и id секрета.
func (r *Repository) LoadBlobBySecret(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (domain.Blob, error) {
	blob, err := scanBlob(r.pool.QueryRow(ctx, loadBlobBySecretQuery, userID, secretID))
	if err != nil {
		return domain.Blob{}, err
	}
	return blob, nil
}

// Delete помечает JSON-секрет удаленным.
func (r *Repository) Delete(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) error {
	var deletedCount int
	if err := r.pool.QueryRow(ctx, deleteSecretQuery, userID, secretID).Scan(&deletedCount); err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	if deletedCount == 0 {
		return usecases.ErrSecretNotFound
	}
	return nil
}

// MarkBlobDeleted помечает blob удаленным без удаления физического файла.
func (r *Repository) MarkBlobDeleted(ctx context.Context, userID uuid.UUID, blobID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, markBlobDeletedQuery, userID, blobID)
	if err != nil {
		return fmt.Errorf("mark blob deleted: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return usecases.ErrBlobNotFound
	}
	return nil
}

// ListBlobsForCleanup возвращает blob-файлы, которые уже удалены логически, но еще лежат на диске.
func (r *Repository) ListBlobsForCleanup(ctx context.Context, before time.Time, limit int) ([]domain.Blob, error) {
	rows, err := r.pool.Query(ctx, listBlobsForCleanupQuery, before, limit)
	if err != nil {
		return nil, fmt.Errorf("select blobs for cleanup: %w", err)
	}
	defer rows.Close()

	result := make([]domain.Blob, 0)
	for rows.Next() {
		blob, err := scanBlob(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, blob)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cleanup blobs: %w", err)
	}
	return result, nil
}

// MarkBlobStorageDeleted помечает физический файл blob удаленным.
func (r *Repository) MarkBlobStorageDeleted(ctx context.Context, userID uuid.UUID, blobID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, markBlobStorageDeletedQuery, userID, blobID)
	if err != nil {
		return fmt.Errorf("mark blob storage deleted: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return usecases.ErrBlobNotFound
	}
	return nil
}

func scanSecret(row pgx.Row) (domain.Secret, error) {
	var secret domain.Secret
	err := row.Scan(
		&secret.ID,
		&secret.UserID,
		&secret.Type,
		&secret.Name,
		&secret.Metadata,
		&secret.MetadataNonce,
		&secret.Payload,
		&secret.PayloadNonce,
		&secret.BlobID,
		&secret.Version,
		&secret.CreatedAt,
		&secret.UpdatedAt,
		&secret.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Secret{}, usecases.ErrSecretNotFound
		}
		return domain.Secret{}, fmt.Errorf("scan secret: %w", err)
	}
	secret.CreatedAt = secret.CreatedAt.UTC()
	secret.UpdatedAt = secret.UpdatedAt.UTC()
	if secret.DeletedAt != nil {
		deletedAt := secret.DeletedAt.UTC()
		secret.DeletedAt = &deletedAt
	}
	return secret, nil
}

func scanBlob(row pgx.Row) (domain.Blob, error) {
	var blob domain.Blob
	err := row.Scan(
		&blob.ID,
		&blob.UserID,
		&blob.OriginalName,
		&blob.StorageName,
		&blob.StoragePath,
		&blob.ContentType,
		&blob.Size,
		&blob.ChecksumSHA256,
		&blob.CreatedAt,
		&blob.DeletedAt,
		&blob.StorageDeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Blob{}, usecases.ErrBlobNotFound
		}
		return domain.Blob{}, fmt.Errorf("scan blob: %w", err)
	}
	blob.CreatedAt = blob.CreatedAt.UTC()
	if blob.DeletedAt != nil {
		deletedAt := blob.DeletedAt.UTC()
		blob.DeletedAt = &deletedAt
	}
	if blob.StorageDeletedAt != nil {
		storageDeletedAt := blob.StorageDeletedAt.UTC()
		blob.StorageDeletedAt = &storageDeletedAt
	}
	return blob, nil
}
