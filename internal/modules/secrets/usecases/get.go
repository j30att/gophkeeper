package usecases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// GetUseCase возвращает JSON-секрет.
type GetUseCase struct {
	repository Repository
	encryptor  Encryptor
}

// NewGetUseCase создает GetUseCase.
func NewGetUseCase(repository Repository, encryptor Encryptor) (*GetUseCase, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: repository", ErrEmptyDependency)
	}
	if encryptor == nil {
		return nil, fmt.Errorf("%w: encryptor", ErrEmptyDependency)
	}
	return &GetUseCase{repository: repository, encryptor: encryptor}, nil
}

// Execute возвращает расшифрованный JSON-секрет пользователя.
func (u *GetUseCase) Execute(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (SecretOutput, error) {
	secret, err := u.repository.Load(ctx, userID, secretID)
	if err != nil {
		return SecretOutput{}, fmt.Errorf("load secret: %w", err)
	}
	output, err := toOutput(secret, u.encryptor)
	if err != nil {
		return SecretOutput{}, err
	}
	if secret.BlobID != nil {
		blob, err := u.repository.LoadBlobBySecret(ctx, userID, secretID)
		if err != nil {
			return SecretOutput{}, fmt.Errorf("load blob: %w", err)
		}
		output.Blob = ptrBlobOutput(toBlobOutput(blob))
	}
	return output, nil
}
