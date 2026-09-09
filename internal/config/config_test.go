package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	t.Run(
		"Должен загрузить значения по умолчанию", func(t *testing.T) {
			t.Setenv("GOPHKEEPER_SERVER_ADDRESS", "")
			t.Setenv("GOPHKEEPER_POSTGRES_DSN", "")
			t.Setenv("GOPHKEEPER_JWT_SECRET", "")
			t.Setenv("GOPHKEEPER_ACCESS_TOKEN_TTL", "")
			t.Setenv("GOPHKEEPER_CRYPTO_MASTER_KEY", "")
			t.Setenv("GOPHKEEPER_BLOB_STORAGE_PATH", "")

			cfg := Load()

			assert.Equal(t, defaultServerAddress, cfg.Server.Address)
			assert.Equal(t, defaultPostgresDSN, cfg.Postgres.DSN)
			assert.Equal(t, defaultJWTSecret, cfg.Auth.JWTSecret)
			assert.Equal(t, defaultTokenTTL, cfg.Auth.AccessTokenTTL)
			assert.Equal(t, defaultCryptoKey, cfg.Crypto.MasterKey)
			assert.Equal(t, defaultBlobStorage, cfg.Storage.BlobPath)
		},
	)

	t.Run(
		"Должен загрузить значения из environment", func(t *testing.T) {
			t.Setenv("GOPHKEEPER_SERVER_ADDRESS", ":9090")
			t.Setenv("GOPHKEEPER_POSTGRES_DSN", "postgres://custom")
			t.Setenv("GOPHKEEPER_JWT_SECRET", "secret")
			t.Setenv("GOPHKEEPER_ACCESS_TOKEN_TTL", "2h")
			t.Setenv("GOPHKEEPER_CRYPTO_MASTER_KEY", "crypto-secret")
			t.Setenv("GOPHKEEPER_BLOB_STORAGE_PATH", "/tmp/gophkeeper-blobs")

			cfg := Load()

			assert.Equal(t, ":9090", cfg.Server.Address)
			assert.Equal(t, "postgres://custom", cfg.Postgres.DSN)
			assert.Equal(t, "secret", cfg.Auth.JWTSecret)
			assert.Equal(t, 2*time.Hour, cfg.Auth.AccessTokenTTL)
			assert.Equal(t, "crypto-secret", cfg.Crypto.MasterKey)
			assert.Equal(t, "/tmp/gophkeeper-blobs", cfg.Storage.BlobPath)
		},
	)

	t.Run(
		"Должен использовать fallback при невалидной duration", func(t *testing.T) {
			t.Setenv("GOPHKEEPER_ACCESS_TOKEN_TTL", "bad")

			cfg := Load()

			assert.Equal(t, defaultTokenTTL, cfg.Auth.AccessTokenTTL)
		},
	)
}
