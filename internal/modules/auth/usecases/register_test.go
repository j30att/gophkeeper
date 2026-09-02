package usecases

import (
	"context"
	"testing"

	"github.com/igor/gophkeeper/internal/modules/auth/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type registerRepositoryMock struct {
	loadUser domain.User
	loadErr  error
	saveUser domain.User
	saveErr  error
}

func (m *registerRepositoryMock) Save(_ context.Context, user domain.User) error {
	m.saveUser = user
	return m.saveErr
}

func (m *registerRepositoryMock) Load(_ context.Context, _ string) (domain.User, error) {
	return m.loadUser, m.loadErr
}

type passwordHasherMock struct {
	hashValue  string
	hashErr    error
	compareErr error
}

func (m *passwordHasherMock) Hash(_ string) (string, error) {
	return m.hashValue, m.hashErr
}

func (m *passwordHasherMock) Compare(_, _ string) error {
	return m.compareErr
}

func TestRegisterUseCase(t *testing.T) {
	t.Run(
		"Тест создания useCase", func(t *testing.T) {
			t.Run(
				"Должен создать useCase без ошибок", func(t *testing.T) {
					repository := &registerRepositoryMock{}
					hasher := &passwordHasherMock{}

					useCase, err := NewRegisterUseCase(repository, hasher)

					require.NoError(t, err)
					assert.Equal(t, repository, useCase.repository)
					assert.Equal(t, hasher, useCase.hasher)
				},
			)

			t.Run(
				"Ошибка, отсутствует зависимость", func(t *testing.T) {
					repository := &registerRepositoryMock{}
					hasher := &passwordHasherMock{}

					tests := []struct {
						repository RegisterRepository
						hasher     PasswordHasher
					}{
						{repository: nil, hasher: hasher},
						{repository: repository, hasher: nil},
					}

					for _, test := range tests {
						_, err := NewRegisterUseCase(test.repository, test.hasher)

						require.Error(t, err)
						assert.ErrorIs(t, err, ErrEmptyDependency)
					}
				},
			)
		},
	)

	t.Run(
		"Тест метода Execute", func(t *testing.T) {
			t.Run(
				"Должен зарегистрировать нового пользователя", func(t *testing.T) {
					repository := &registerRepositoryMock{loadErr: ErrUserNotFound}
					hasher := &passwordHasherMock{hashValue: "hashed-password"}
					useCase, err := NewRegisterUseCase(repository, hasher)
					require.NoError(t, err)

					output, err := useCase.Execute(
						context.Background(),
						RegisterInput{Login: "igor", Password: "password-1"},
					)

					require.NoError(t, err)
					assert.NotEmpty(t, output.UserID)
					assert.Equal(t, output.UserID, repository.saveUser.ID)
					assert.Equal(t, "igor", repository.saveUser.Login)
					assert.Equal(t, "hashed-password", repository.saveUser.PasswordHash)
					assert.False(t, repository.saveUser.CreatedAt.IsZero())
					assert.False(t, repository.saveUser.UpdatedAt.IsZero())
				},
			)

			t.Run(
				"Должен вернуть ошибку, пользователь уже существует", func(t *testing.T) {
					repository := &registerRepositoryMock{loadUser: domain.User{Login: "igor"}}
					hasher := &passwordHasherMock{hashValue: "hashed-password"}
					useCase, err := NewRegisterUseCase(repository, hasher)
					require.NoError(t, err)

					_, err = useCase.Execute(
						context.Background(),
						RegisterInput{Login: "igor", Password: "password-1"},
					)

					require.Error(t, err)
					assert.ErrorIs(t, err, ErrUserAlreadyExists)
				},
			)

			t.Run(
				"Должен вернуть ошибку загрузки пользователя", func(t *testing.T) {
					repository := &registerRepositoryMock{loadErr: assert.AnError}
					hasher := &passwordHasherMock{hashValue: "hashed-password"}
					useCase, err := NewRegisterUseCase(repository, hasher)
					require.NoError(t, err)

					_, err = useCase.Execute(
						context.Background(),
						RegisterInput{Login: "igor", Password: "password-1"},
					)

					require.Error(t, err)
					assert.ErrorIs(t, err, assert.AnError)
				},
			)

			t.Run(
				"Должен вернуть ошибку хеширования", func(t *testing.T) {
					repository := &registerRepositoryMock{loadErr: ErrUserNotFound}
					hasher := &passwordHasherMock{hashErr: assert.AnError}
					useCase, err := NewRegisterUseCase(repository, hasher)
					require.NoError(t, err)

					_, err = useCase.Execute(
						context.Background(),
						RegisterInput{Login: "igor", Password: "password-1"},
					)

					require.Error(t, err)
					assert.ErrorIs(t, err, assert.AnError)
				},
			)

			t.Run(
				"Должен вернуть ошибку сохранения пользователя", func(t *testing.T) {
					repository := &registerRepositoryMock{loadErr: ErrUserNotFound, saveErr: assert.AnError}
					hasher := &passwordHasherMock{hashValue: "hashed-password"}
					useCase, err := NewRegisterUseCase(repository, hasher)
					require.NoError(t, err)

					_, err = useCase.Execute(
						context.Background(),
						RegisterInput{Login: "igor", Password: "password-1"},
					)

					require.Error(t, err)
					assert.ErrorIs(t, err, assert.AnError)
				},
			)
		},
	)
}
