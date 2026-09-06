package usecases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ListUseCase возвращает список JSON-секретов.
type ListUseCase struct {
	repository Repository
	encryptor  Encryptor
}

// NewListUseCase создает ListUseCase.
func NewListUseCase(repository Repository, encryptor Encryptor) (*ListUseCase, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: repository", ErrEmptyDependency)
	}
	if encryptor == nil {
		return nil, fmt.Errorf("%w: encryptor", ErrEmptyDependency)
	}
	return &ListUseCase{repository: repository, encryptor: encryptor}, nil
}

// Execute возвращает расшифрованный список JSON-секретов пользователя без payload.
func (u *ListUseCase) Execute(ctx context.Context, userID uuid.UUID) ([]SecretListItem, error) {
	secrets, err := u.repository.List(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	result := make([]SecretListItem, 0, len(secrets))
	for _, secret := range secrets {
		item, err := toListItem(secret, u.encryptor)
		if err != nil {
			return nil, err
		}
		if secret.BlobID != nil {
			blob, err := u.repository.LoadBlobBySecret(ctx, userID, secret.ID)
			if err != nil {
				return nil, fmt.Errorf("load blob: %w", err)
			}
			item.Blob = ptrBlobOutput(toBlobOutput(blob))
		}
		result = append(result, item)
	}
	return result, nil
}
