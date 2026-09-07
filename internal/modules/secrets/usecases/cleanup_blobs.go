package usecases

import (
	"context"
	"fmt"
	"time"
)

const defaultCleanupLimit = 100

// CleanupBlobsUseCase удаляет с диска файлы blob-секретов, которые уже удалены логически.
type CleanupBlobsUseCase struct {
	repository Repository
	storage    BlobStorage
}

// CleanupBlobsInput описывает параметры уборки blob-файлов.
type CleanupBlobsInput struct {
	Before time.Time
	Limit  int
}

// CleanupBlobsOutput описывает результат уборки blob-файлов.
type CleanupBlobsOutput struct {
	Deleted int
}

// NewCleanupBlobsUseCase создает CleanupBlobsUseCase.
func NewCleanupBlobsUseCase(repository Repository, storage BlobStorage) (*CleanupBlobsUseCase, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: repository", ErrEmptyDependency)
	}
	if storage == nil {
		return nil, fmt.Errorf("%w: storage", ErrEmptyDependency)
	}
	return &CleanupBlobsUseCase{
		repository: repository,
		storage:    storage,
	}, nil
}

// Execute удаляет найденные файлы и помечает их как удаленные из физического хранилища.
func (u *CleanupBlobsUseCase) Execute(ctx context.Context, input CleanupBlobsInput) (CleanupBlobsOutput, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = defaultCleanupLimit
	}
	before := input.Before
	if before.IsZero() {
		before = time.Now().UTC()
	}

	blobs, err := u.repository.ListBlobsForCleanup(ctx, before, limit)
	if err != nil {
		return CleanupBlobsOutput{}, err
	}

	output := CleanupBlobsOutput{}
	for _, blob := range blobs {
		if err = u.storage.Delete(ctx, blob.StorageName); err != nil {
			return output, fmt.Errorf("delete blob content %s: %w", blob.ID, err)
		}
		if err = u.repository.MarkBlobStorageDeleted(ctx, blob.UserID, blob.ID); err != nil {
			return output, fmt.Errorf("mark blob storage deleted %s: %w", blob.ID, err)
		}
		output.Deleted++
	}
	return output, nil
}
