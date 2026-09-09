package usecases

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
)

// UpdateBlobContentUseCase заменяет содержимое существующего blob-секрета.
type UpdateBlobContentUseCase struct {
	repository Repository
	encryptor  Encryptor
	storage    BlobStorage
}

// NewUpdateBlobContentUseCase создает UpdateBlobContentUseCase.
func NewUpdateBlobContentUseCase(
	repository Repository,
	encryptor Encryptor,
	storage BlobStorage,
) (*UpdateBlobContentUseCase, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: repository", ErrEmptyDependency)
	}
	if encryptor == nil {
		return nil, fmt.Errorf("%w: encryptor", ErrEmptyDependency)
	}
	if storage == nil {
		return nil, fmt.Errorf("%w: storage", ErrEmptyDependency)
	}
	return &UpdateBlobContentUseCase{repository: repository, encryptor: encryptor, storage: storage}, nil
}

// Execute потоково шифрует новое содержимое и переключает secret на новый blob.
func (u *UpdateBlobContentUseCase) Execute(ctx context.Context, input BlobContentInput) (SecretOutput, error) {
	if input.ExpectedVersion < 1 {
		return SecretOutput{}, ErrSecretVersionConflict
	}
	if input.Content == nil {
		return SecretOutput{}, ErrEmptyContent
	}
	current, err := u.repository.Load(ctx, input.UserID, input.ID)
	if err != nil {
		return SecretOutput{}, fmt.Errorf("load secret: %w", err)
	}
	if !domain.IsBlobType(current.Type) || current.BlobID == nil {
		return SecretOutput{}, ErrInvalidSecretType
	}
	if current.Version != input.ExpectedVersion {
		return SecretOutput{}, ErrSecretVersionConflict
	}

	oldBlob, err := u.repository.LoadBlobBySecret(ctx, input.UserID, input.ID)
	if err != nil {
		return SecretOutput{}, fmt.Errorf("load blob: %w", err)
	}
	newBlob, err := u.saveBlob(ctx, input.UserID, input.OriginalName, input.ContentType, input.Content)
	if err != nil {
		return SecretOutput{}, err
	}
	if err = u.repository.SaveBlob(ctx, newBlob); err != nil {
		return SecretOutput{}, fmt.Errorf("save blob metadata: %w", err)
	}

	secret := current
	secret.BlobID = &newBlob.ID
	secret.Version = current.Version + 1
	secret.UpdatedAt = time.Now().UTC()
	if err = u.repository.Update(ctx, secret, input.ExpectedVersion); err != nil {
		_ = u.repository.MarkBlobDeleted(ctx, input.UserID, newBlob.ID)
		return SecretOutput{}, fmt.Errorf("update secret blob: %w", err)
	}
	if err = u.repository.MarkBlobDeleted(ctx, input.UserID, oldBlob.ID); err != nil {
		return SecretOutput{}, fmt.Errorf("mark old blob deleted: %w", err)
	}

	output, err := toOutput(secret, u.encryptor)
	if err != nil {
		return SecretOutput{}, err
	}
	output.Blob = ptrBlobOutput(toBlobOutput(newBlob))
	return output, nil
}

func (u *UpdateBlobContentUseCase) saveBlob(
	ctx context.Context,
	userID uuid.UUID,
	originalName string,
	contentType string,
	content io.Reader,
) (domain.Blob, error) {
	now := time.Now().UTC()
	blobID := uuid.New()
	storageName := blobID.String() + ".gpk"
	reader, writer := io.Pipe()
	encryptResult := make(chan encryptStreamResult, 1)
	go func() {
		size, checksum, encryptErr := u.encryptor.EncryptStream(content, writer)
		closeErr := writer.Close()
		if encryptErr == nil {
			encryptErr = closeErr
		}
		encryptResult <- encryptStreamResult{size: size, checksumSHA256: checksum, err: encryptErr}
	}()

	storagePath, err := u.storage.Save(ctx, storageName, reader)
	if err != nil {
		_ = reader.Close()
		return domain.Blob{}, fmt.Errorf("save blob content: %w", err)
	}
	result := <-encryptResult
	if result.err != nil {
		return domain.Blob{}, fmt.Errorf("encrypt blob content: %w", result.err)
	}

	return domain.Blob{
		ID:             blobID,
		UserID:         userID,
		OriginalName:   normalizeOriginalName(originalName),
		StorageName:    storageName,
		StoragePath:    storagePath,
		ContentType:    normalizeContentType(contentType),
		Size:           result.size,
		ChecksumSHA256: result.checksumSHA256,
		CreatedAt:      now,
	}, nil
}
