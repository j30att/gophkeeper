package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
)

// UpdateUseCase обновляет JSON-секрет.
type UpdateUseCase struct {
	repository Repository
	encryptor  Encryptor
}

// NewUpdateUseCase создает UpdateUseCase.
func NewUpdateUseCase(repository Repository, encryptor Encryptor) (*UpdateUseCase, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: repository", ErrEmptyDependency)
	}
	if encryptor == nil {
		return nil, fmt.Errorf("%w: encryptor", ErrEmptyDependency)
	}
	return &UpdateUseCase{repository: repository, encryptor: encryptor}, nil
}

// Execute обновляет JSON-секрет пользователя.
func (u *UpdateUseCase) Execute(ctx context.Context, input UpdateSecretInput) (SecretOutput, error) {
	if err := validateStructuredInput(input.Type, input.Metadata, input.Payload); err != nil {
		return SecretOutput{}, err
	}
	current, err := u.repository.Load(ctx, input.UserID, input.ID)
	if err != nil {
		return SecretOutput{}, fmt.Errorf("load secret: %w", err)
	}
	metadata, metadataNonce, err := encryptJSON(u.encryptor, input.Metadata, []byte("{}"))
	if err != nil {
		return SecretOutput{}, fmt.Errorf("encrypt metadata: %w", err)
	}
	payload, payloadNonce, err := encryptJSON(u.encryptor, input.Payload, nil)
	if err != nil {
		return SecretOutput{}, fmt.Errorf("encrypt payload: %w", err)
	}

	secret := domain.Secret{
		ID:            input.ID,
		UserID:        input.UserID,
		Type:          input.Type,
		Name:          input.Name,
		Metadata:      metadata,
		MetadataNonce: metadataNonce,
		Payload:       payload,
		PayloadNonce:  payloadNonce,
		Version:       current.Version + 1,
		CreatedAt:     current.CreatedAt,
		UpdatedAt:     time.Now().UTC(),
	}
	if err = u.repository.Update(ctx, secret); err != nil {
		return SecretOutput{}, fmt.Errorf("update secret: %w", err)
	}
	return toOutput(secret, u.encryptor)
}
