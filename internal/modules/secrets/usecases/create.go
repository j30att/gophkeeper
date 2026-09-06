package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
)

// CreateUseCase создает JSON-секрет.
type CreateUseCase struct {
	repository Repository
	encryptor  Encryptor
}

// NewCreateUseCase создает CreateUseCase.
func NewCreateUseCase(repository Repository, encryptor Encryptor) (*CreateUseCase, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: repository", ErrEmptyDependency)
	}
	if encryptor == nil {
		return nil, fmt.Errorf("%w: encryptor", ErrEmptyDependency)
	}
	return &CreateUseCase{repository: repository, encryptor: encryptor}, nil
}

// Execute создает JSON-секрет пользователя.
func (u *CreateUseCase) Execute(ctx context.Context, input SecretInput) (SecretOutput, error) {
	if err := validateStructuredInput(input.Type, input.Metadata, input.Payload); err != nil {
		return SecretOutput{}, err
	}
	metadata, metadataNonce, err := encryptJSON(u.encryptor, input.Metadata, []byte("{}"))
	if err != nil {
		return SecretOutput{}, fmt.Errorf("encrypt metadata: %w", err)
	}
	payload, payloadNonce, err := encryptJSON(u.encryptor, input.Payload, nil)
	if err != nil {
		return SecretOutput{}, fmt.Errorf("encrypt payload: %w", err)
	}

	now := time.Now().UTC()
	secret := domain.Secret{
		ID:            uuid.New(),
		UserID:        input.UserID,
		Type:          input.Type,
		Name:          input.Name,
		Metadata:      metadata,
		MetadataNonce: metadataNonce,
		Payload:       payload,
		PayloadNonce:  payloadNonce,
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err = u.repository.Save(ctx, secret); err != nil {
		return SecretOutput{}, fmt.Errorf("save secret: %w", err)
	}
	return toOutput(secret, u.encryptor)
}
