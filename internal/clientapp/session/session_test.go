package session

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore(t *testing.T) {
	t.Run("Должен сохранить и загрузить session", func(t *testing.T) {
		store, err := NewStore(filepath.Join(t.TempDir(), "session.json"))
		require.NoError(t, err)

		err = store.Save(Session{Token: "token"})
		require.NoError(t, err)
		loaded, err := store.Load()

		require.NoError(t, err)
		assert.Equal(t, "token", loaded.Token)
	})

	t.Run("Должен вернуть ErrNotFound без файла", func(t *testing.T) {
		store, err := NewStore(filepath.Join(t.TempDir(), "session.json"))
		require.NoError(t, err)

		_, err = store.Load()

		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("Должен удалить session", func(t *testing.T) {
		store, err := NewStore(filepath.Join(t.TempDir(), "session.json"))
		require.NoError(t, err)
		require.NoError(t, store.Save(Session{Token: "token"}))

		require.NoError(t, store.Delete())
		_, err = store.Load()

		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("Должен игнорировать удаление отсутствующей session", func(t *testing.T) {
		store, err := NewStore(filepath.Join(t.TempDir(), "session.json"))
		require.NoError(t, err)

		require.NoError(t, store.Delete())
	})

	t.Run("Должен вернуть ошибку без path", func(t *testing.T) {
		_, err := NewStore("")

		require.Error(t, err)
	})

	t.Run("Должен вернуть ошибку при пустом token", func(t *testing.T) {
		store, err := NewStore(filepath.Join(t.TempDir(), "session.json"))
		require.NoError(t, err)

		err = store.Save(Session{})

		require.Error(t, err)
	})
}
