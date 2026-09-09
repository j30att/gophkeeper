package cache

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/igor/gophkeeper/internal/clientapp/api"
)

func TestStore(t *testing.T) {
	t.Run("Должен сохранить и загрузить cache", func(t *testing.T) {
		store, err := NewStore(filepath.Join(t.TempDir(), "secrets.json"))
		require.NoError(t, err)
		now := time.Date(2026, 9, 9, 1, 2, 3, 0, time.UTC)
		secretID := uuid.New()

		err = store.Save(Cache{
			LastSyncAt: &now,
			Secrets: []api.Secret{
				{ID: secretID, Name: "github", Type: api.SecretTypeCredentials, Version: 1},
			},
		})
		require.NoError(t, err)
		loaded, err := store.Load()

		require.NoError(t, err)
		require.Len(t, loaded.Secrets, 1)
		assert.Equal(t, secretID, loaded.Secrets[0].ID)
		require.NotNil(t, loaded.LastSyncAt)
		assert.Equal(t, now, *loaded.LastSyncAt)
	})

	t.Run("Должен вернуть ErrNotFound без файла", func(t *testing.T) {
		store, err := NewStore(filepath.Join(t.TempDir(), "secrets.json"))
		require.NoError(t, err)

		_, err = store.Load()

		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("Должен удалить cache", func(t *testing.T) {
		store, err := NewStore(filepath.Join(t.TempDir(), "secrets.json"))
		require.NoError(t, err)
		require.NoError(t, store.Save(Cache{}))

		require.NoError(t, store.Delete())
		_, err = store.Load()

		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("Должен игнорировать удаление отсутствующего cache", func(t *testing.T) {
		store, err := NewStore(filepath.Join(t.TempDir(), "secrets.json"))
		require.NoError(t, err)

		require.NoError(t, store.Delete())
	})

	t.Run("Должен вернуть ошибку без path", func(t *testing.T) {
		_, err := NewStore("")

		require.Error(t, err)
	})
}
