package filesystem

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlobStorage(t *testing.T) {
	t.Run("Должен сохранить и открыть blob", func(t *testing.T) {
		storage, err := NewBlobStorage(t.TempDir())
		require.NoError(t, err)

		path, err := storage.Save(context.Background(), "file.gpk", strings.NewReader("content"))
		require.NoError(t, err)
		assert.NotEmpty(t, path)

		file, err := storage.Open(context.Background(), "file.gpk")
		require.NoError(t, err)
		defer file.Close()

		content, err := io.ReadAll(file)
		require.NoError(t, err)
		assert.Equal(t, "content", string(content))
	})

	t.Run("Должен вернуть ошибку при path traversal", func(t *testing.T) {
		storage, err := NewBlobStorage(t.TempDir())
		require.NoError(t, err)

		_, err = storage.Save(context.Background(), "../file.gpk", strings.NewReader("content"))

		require.Error(t, err)
	})
}
