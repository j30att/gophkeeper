package usecases

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/igor/gophkeeper/internal/modules/auth/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type loginRepositoryMock struct {
	user domain.User
	err  error
}

func (m *loginRepositoryMock) Load(_ context.Context, _ string) (domain.User, error) {
	return m.user, m.err
}

type tokenIssuerMock struct {
	token string
	err   error
}

func (m *tokenIssuerMock) Issue(_ uuid.UUID, _ string) (string, error) {
	return m.token, m.err
}

func TestLoginUseCase(t *testing.T) {
	t.Run(
		"Тест создания useCase", func(t *testing.T) {
			t.Run(
				"Должен создать useCase без ошибок", func(t *testing.T) {
					repository := &loginRepositoryMock{}
					hasher := &passwordHasherMock{}
					token := &tokenIssuerMock{}

					useCase, err := NewLoginUseCase(repository, hasher, token)

					require.NoError(t, err)
					assert.Equal(t, repository, useCase.repository)
					assert.Equal(t, hasher, useCase.hasher)
					assert.Equal(t, token, useCase.token)
				},
			)

			t.Run(
				"Ошибка, отсутствует зависимость", func(t *testing.T) {
					repository := &loginRepositoryMock{}
					hasher := &passwordHasherMock{}
					token := &tokenIssuerMock{}

					tests := []struct {
						repository LoginRepository
						hasher     PasswordHasher
						token      TokenIssuer
					}{
						{repository: nil, hasher: hasher, token: token},
						{repository: repository, hasher: nil, token: token},
						{repository: repository, hasher: hasher, token: nil},
					}

					for _, test := range tests {
						_, err := NewLoginUseCase(test.repository, test.hasher, test.token)

						require.Error(t, err)
						assert.ErrorIs(t, err, ErrEmptyDependency)
					}
				},
			)
		},
	)

	t.Run(
		"Тест метода Execute", func(t *testing.T) {
			userID := uuid.New()

			t.Run(
				"Должен вернуть access token", func(t *testing.T) {
					repository := &loginRepositoryMock{
						user: domain.User{ID: userID, Login: "igor", PasswordHash: "hashed-password"},
					}
					hasher := &passwordHasherMock{}
					token := &tokenIssuerMock{token: "jwt-token"}
					useCase, err := NewLoginUseCase(repository, hasher, token)
					require.NoError(t, err)

					output, err := useCase.Execute(
						context.Background(),
						LoginInput{Login: "igor", Password: "password-1"},
					)

					require.NoError(t, err)
					assert.Equal(t, "jwt-token", output.Token)
				},
			)

			t.Run(
				"Должен скрыть отсутствие пользователя как invalid credentials", func(t *testing.T) {
					repository := &loginRepositoryMock{err: ErrUserNotFound}
					hasher := &passwordHasherMock{}
					token := &tokenIssuerMock{token: "jwt-token"}
					useCase, err := NewLoginUseCase(repository, hasher, token)
					require.NoError(t, err)

					_, err = useCase.Execute(
						context.Background(),
						LoginInput{Login: "igor", Password: "password-1"},
					)

					require.Error(t, err)
					assert.ErrorIs(t, err, ErrInvalidCredentials)
				},
			)

			t.Run(
				"Должен вернуть ошибку загрузки пользователя", func(t *testing.T) {
					repository := &loginRepositoryMock{err: assert.AnError}
					hasher := &passwordHasherMock{}
					token := &tokenIssuerMock{token: "jwt-token"}
					useCase, err := NewLoginUseCase(repository, hasher, token)
					require.NoError(t, err)

					_, err = useCase.Execute(
						context.Background(),
						LoginInput{Login: "igor", Password: "password-1"},
					)

					require.Error(t, err)
					assert.ErrorIs(t, err, assert.AnError)
				},
			)

			t.Run(
				"Должен вернуть invalid credentials при неверном пароле", func(t *testing.T) {
					repository := &loginRepositoryMock{
						user: domain.User{ID: userID, Login: "igor", PasswordHash: "hashed-password"},
					}
					hasher := &passwordHasherMock{compareErr: assert.AnError}
					token := &tokenIssuerMock{token: "jwt-token"}
					useCase, err := NewLoginUseCase(repository, hasher, token)
					require.NoError(t, err)

					_, err = useCase.Execute(
						context.Background(),
						LoginInput{Login: "igor", Password: "wrong-password"},
					)

					require.Error(t, err)
					assert.ErrorIs(t, err, ErrInvalidCredentials)
				},
			)

			t.Run(
				"Должен вернуть ошибку выпуска token", func(t *testing.T) {
					repository := &loginRepositoryMock{
						user: domain.User{ID: userID, Login: "igor", PasswordHash: "hashed-password"},
					}
					hasher := &passwordHasherMock{}
					token := &tokenIssuerMock{err: assert.AnError}
					useCase, err := NewLoginUseCase(repository, hasher, token)
					require.NoError(t, err)

					_, err = useCase.Execute(
						context.Background(),
						LoginInput{Login: "igor", Password: "password-1"},
					)

					require.Error(t, err)
					assert.ErrorIs(t, err, assert.AnError)
				},
			)
		},
	)
}
