package usecases

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/igor/gophkeeper/internal/modules/auth/domain"
)

// LoginRepository загружает пользователей для аутентификации.
type LoginRepository interface {
	Load(ctx context.Context, login string) (domain.User, error)
}

// TokenIssuer создает access tokens для аутентифицированных пользователей.
type TokenIssuer interface {
	Issue(userID uuid.UUID, login string) (string, error)
}

// LoginInput содержит данные команды логина.
type LoginInput struct {
	Login    string
	Password string
}

// LoginOutput содержит результат логина.
type LoginOutput struct {
	Token string
}

// LoginUseCase аутентифицирует пользователя и выпускает access token.
type LoginUseCase struct {
	repository LoginRepository
	hasher     PasswordHasher
	token      TokenIssuer
}

// NewLoginUseCase создает LoginUseCase.
func NewLoginUseCase(repository LoginRepository, hasher PasswordHasher, token TokenIssuer) (*LoginUseCase, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: repository", ErrEmptyDependency)
	}
	if hasher == nil {
		return nil, fmt.Errorf("%w: hasher", ErrEmptyDependency)
	}
	if token == nil {
		return nil, fmt.Errorf("%w: token", ErrEmptyDependency)
	}
	return &LoginUseCase{repository: repository, hasher: hasher, token: token}, nil
}

// Execute аутентифицирует пользователя и возвращает JWT access token.
func (u *LoginUseCase) Execute(ctx context.Context, input LoginInput) (LoginOutput, error) {
	user, err := u.repository.Load(ctx, input.Login)
	if err != nil {
		if err == ErrUserNotFound {
			return LoginOutput{}, ErrInvalidCredentials
		}
		return LoginOutput{}, fmt.Errorf("load user: %w", err)
	}

	if err = u.hasher.Compare(user.PasswordHash, input.Password); err != nil {
		return LoginOutput{}, ErrInvalidCredentials
	}

	accessToken, err := u.token.Issue(user.ID, user.Login)
	if err != nil {
		return LoginOutput{}, fmt.Errorf("issue token: %w", err)
	}

	return LoginOutput{Token: accessToken}, nil
}
