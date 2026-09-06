package usecases

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
)

// GetBlobContentUseCase возвращает расшифрованное содержимое blob-секрета.
type GetBlobContentUseCase struct {
	repository Repository
	encryptor  Encryptor
	storage    BlobStorage
}

// NewGetBlobContentUseCase создает GetBlobContentUseCase.
func NewGetBlobContentUseCase(repository Repository, encryptor Encryptor, storage BlobStorage) (*GetBlobContentUseCase, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: repository", ErrEmptyDependency)
	}
	if encryptor == nil {
		return nil, fmt.Errorf("%w: encryptor", ErrEmptyDependency)
	}
	if storage == nil {
		return nil, fmt.Errorf("%w: storage", ErrEmptyDependency)
	}
	return &GetBlobContentUseCase{repository: repository, encryptor: encryptor, storage: storage}, nil
}

// Execute открывает blob-файл и возвращает reader с расшифрованным содержимым.
func (u *GetBlobContentUseCase) Execute(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (BlobContentOutput, error) {
	blob, err := u.repository.LoadBlobBySecret(ctx, userID, secretID)
	if err != nil {
		return BlobContentOutput{}, fmt.Errorf("load blob: %w", err)
	}

	encryptedContent, err := u.storage.Open(ctx, blob.StorageName)
	if err != nil {
		return BlobContentOutput{}, fmt.Errorf("open blob content: %w", err)
	}

	reader, writer := io.Pipe()
	go func() {
		defer encryptedContent.Close()
		err := u.encryptor.DecryptStream(encryptedContent, writer)
		if err != nil {
			_ = writer.CloseWithError(err)
			return
		}
		_ = writer.Close()
	}()

	return BlobContentOutput{
		Blob:    toBlobOutput(blob),
		Content: reader,
	}, nil
}
