package usecases

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
)

// CreateBlobUseCase создает секрет, содержимое которого хранится в blob-файле.
type CreateBlobUseCase struct {
	repository Repository
	encryptor  Encryptor
	storage    BlobStorage
}

// NewCreateBlobUseCase создает CreateBlobUseCase.
func NewCreateBlobUseCase(repository Repository, encryptor Encryptor, storage BlobStorage) (*CreateBlobUseCase, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: repository", ErrEmptyDependency)
	}
	if encryptor == nil {
		return nil, fmt.Errorf("%w: encryptor", ErrEmptyDependency)
	}
	if storage == nil {
		return nil, fmt.Errorf("%w: storage", ErrEmptyDependency)
	}
	return &CreateBlobUseCase{repository: repository, encryptor: encryptor, storage: storage}, nil
}

// Execute создает blob-секрет и сохраняет содержимое потоково.
func (u *CreateBlobUseCase) Execute(ctx context.Context, input BlobSecretInput) (SecretOutput, error) {
	if input.Content == nil {
		return SecretOutput{}, ErrEmptyContent
	}
	if err := validateBlobInput(input.Type, input.Metadata); err != nil {
		return SecretOutput{}, err
	}

	metadata, metadataNonce, err := encryptJSON(u.encryptor, input.Metadata, []byte("{}"))
	if err != nil {
		return SecretOutput{}, fmt.Errorf("encrypt metadata: %w", err)
	}

	now := time.Now().UTC()
	blobID := uuid.New()
	storageName := blobID.String() + ".gpk"
	reader, writer := io.Pipe()
	encryptResult := make(chan encryptStreamResult, 1)
	go func() {
		size, checksum, encryptErr := u.encryptor.EncryptStream(input.Content, writer)
		closeErr := writer.Close()
		if encryptErr == nil {
			encryptErr = closeErr
		}
		encryptResult <- encryptStreamResult{size: size, checksumSHA256: checksum, err: encryptErr}
	}()

	storagePath, err := u.storage.Save(ctx, storageName, reader)
	if err != nil {
		_ = reader.Close()
		return SecretOutput{}, fmt.Errorf("save blob content: %w", err)
	}
	result := <-encryptResult
	if result.err != nil {
		return SecretOutput{}, fmt.Errorf("encrypt blob content: %w", result.err)
	}

	blob := domain.Blob{
		ID:             blobID,
		UserID:         input.UserID,
		OriginalName:   normalizeOriginalName(input.OriginalName),
		StorageName:    storageName,
		StoragePath:    storagePath,
		ContentType:    normalizeContentType(input.ContentType),
		Size:           result.size,
		ChecksumSHA256: result.checksumSHA256,
		CreatedAt:      now,
	}
	if err = u.repository.SaveBlob(ctx, blob); err != nil {
		return SecretOutput{}, fmt.Errorf("save blob metadata: %w", err)
	}

	payload, payloadNonce, err := encryptJSON(u.encryptor, nil, []byte("{}"))
	if err != nil {
		return SecretOutput{}, fmt.Errorf("encrypt empty payload: %w", err)
	}
	secret := domain.Secret{
		ID:            uuid.New(),
		UserID:        input.UserID,
		Type:          input.Type,
		Name:          input.Name,
		Metadata:      metadata,
		MetadataNonce: metadataNonce,
		Payload:       payload,
		PayloadNonce:  payloadNonce,
		BlobID:        &blobID,
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err = u.repository.Save(ctx, secret); err != nil {
		return SecretOutput{}, fmt.Errorf("save secret: %w", err)
	}

	output, err := toOutput(secret, u.encryptor)
	if err != nil {
		return SecretOutput{}, err
	}
	output.Blob = ptrBlobOutput(toBlobOutput(blob))
	return output, nil
}

type encryptStreamResult struct {
	size           int64
	checksumSHA256 string
	err            error
}

func normalizeOriginalName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == "/" || base == "" {
		return "content"
	}
	return base
}

func normalizeContentType(contentType string) string {
	if strings.TrimSpace(contentType) == "" {
		return "application/octet-stream"
	}
	return contentType
}

func ptrBlobOutput(blob BlobOutput) *BlobOutput {
	return &blob
}
