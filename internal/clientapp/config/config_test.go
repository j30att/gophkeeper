package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("Должен использовать server URL из аргумента", func(t *testing.T) {
		cfg, err := Load("http://localhost:9090/")

		require.NoError(t, err)
		assert.Equal(t, "http://localhost:9090", cfg.ServerURL)
		assert.Contains(t, cfg.SessionPath, ".gophkeeper")
	})

	t.Run("Должен использовать server URL из environment", func(t *testing.T) {
		t.Setenv("GOPHKEEPER_SERVER_URL", "http://example.com/")

		cfg, err := Load("")

		require.NoError(t, err)
		assert.Equal(t, "http://example.com", cfg.ServerURL)
	})

	t.Run("Должен использовать default server URL", func(t *testing.T) {
		t.Setenv("GOPHKEEPER_SERVER_URL", "")

		cfg, err := Load("")

		require.NoError(t, err)
		assert.Equal(t, defaultServerURL, cfg.ServerURL)
	})

	t.Run("Должен вернуть ошибку без home dir", func(t *testing.T) {
		oldHome := os.Getenv("HOME")
		t.Setenv("HOME", "")
		t.Cleanup(func() { _ = os.Setenv("HOME", oldHome) })

		_, err := Load("")

		require.Error(t, err)
	})
}
