package usecases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// DeleteUseCase удаляет JSON-секрет.
type DeleteUseCase struct {
	repository Repository
}

// NewDeleteUseCase создает DeleteUseCase.
func NewDeleteUseCase(repository Repository) (*DeleteUseCase, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: repository", ErrEmptyDependency)
	}
	return &DeleteUseCase{repository: repository}, nil
}

// Execute выполняет soft-delete JSON-секрета пользователя.
func (u *DeleteUseCase) Execute(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) error {
	if err := u.repository.Delete(ctx, userID, secretID); err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	return nil
}
