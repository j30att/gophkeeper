package usecases

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/igor/gophkeeper/internal/modules/auth/domain"
)

// RegisterRepository сохраняет и загружает пользователей для регистрации.
type RegisterRepository interface {
	Save(ctx context.Context, user domain.User) error
	Load(ctx context.Context, login string) (domain.User, error)
}

// PasswordHasher хеширует и проверяет пароли пользователей.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// RegisterInput содержит данные команды регистрации.
type RegisterInput struct {
	Login    string
	Password string
}

// RegisterOutput содержит результат регистрации.
type RegisterOutput struct {
	UserID uuid.UUID
}

// RegisterUseCase создает новую учетную запись пользователя.
type RegisterUseCase struct {
	repository RegisterRepository
	hasher     PasswordHasher
}

// NewRegisterUseCase создает RegisterUseCase.
func NewRegisterUseCase(repository RegisterRepository, hasher PasswordHasher) (*RegisterUseCase, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: repository", ErrEmptyDependency)
	}
	if hasher == nil {
		return nil, fmt.Errorf("%w: hasher", ErrEmptyDependency)
	}
	return &RegisterUseCase{repository: repository, hasher: hasher}, nil
}

// Execute регистрирует пользователя с уникальным login.
func (u *RegisterUseCase) Execute(ctx context.Context, input RegisterInput) (RegisterOutput, error) {
	_, err := u.repository.Load(ctx, input.Login)
	if err == nil {
		return RegisterOutput{}, ErrUserAlreadyExists
	}
	if !errors.Is(err, ErrUserNotFound) {
		return RegisterOutput{}, fmt.Errorf("load user: %w", err)
	}

	passwordHash, err := u.hasher.Hash(input.Password)
	if err != nil {
		return RegisterOutput{}, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	user := domain.User{
		ID:           uuid.New(),
		Login:        input.Login,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err = u.repository.Save(ctx, user); err != nil {
		return RegisterOutput{}, fmt.Errorf("save user: %w", err)
	}

	return RegisterOutput{UserID: user.ID}, nil
}
