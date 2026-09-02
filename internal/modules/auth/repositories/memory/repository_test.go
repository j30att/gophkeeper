package memory

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/igor/gophkeeper/internal/modules/auth/domain"
	"github.com/igor/gophkeeper/internal/modules/auth/usecases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository(t *testing.T) {
	t.Run(
		"Должен создать репозиторий", func(t *testing.T) {
			repository := NewUserRepository()

			require.NotNil(t, repository)
			assert.NotNil(t, repository.byLogin)
		},
	)

	t.Run(
		"Должен сохранить и загрузить пользователя", func(t *testing.T) {
			repository := NewUserRepository()
			user := domain.User{ID: uuid.New(), Login: "igor", PasswordHash: "hash"}

			err := repository.Save(context.Background(), user)
			require.NoError(t, err)

			loaded, err := repository.Load(context.Background(), user.Login)

			require.NoError(t, err)
			assert.Equal(t, user, loaded)
		},
	)

	t.Run(
		"Должен вернуть ошибку, пользователь не найден", func(t *testing.T) {
			repository := NewUserRepository()

			_, err := repository.Load(context.Background(), "missing")

			require.Error(t, err)
			assert.ErrorIs(t, err, usecases.ErrUserNotFound)
		},
	)
}
