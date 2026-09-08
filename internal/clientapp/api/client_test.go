package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	t.Run("Должен выполнить login", func(t *testing.T) {
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/auth/login", r.URL.Path)
				_ = json.NewEncoder(w).Encode(map[string]string{"token": "jwt"})
			}),
		)
		defer server.Close()
		client, err := NewClient(server.URL, server.Client())
		require.NoError(t, err)

		token, err := client.Login(context.Background(), "igor", "password-1")

		require.NoError(t, err)
		assert.Equal(t, "jwt", token)
	})

	t.Run("Должен выполнить sync", func(t *testing.T) {
		since := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer jwt", r.Header.Get("Authorization"))
				assert.Equal(t, since.Format(time.RFC3339), r.URL.Query().Get("since"))
				_ = json.NewEncoder(w).Encode(
					SyncResponse{
						ServerTime: since,
						Secrets:    []Secret{{Name: "github", Type: SecretTypeCredentials}},
					},
				)
			}),
		)
		defer server.Close()
		client, err := NewClient(server.URL, server.Client())
		require.NoError(t, err)

		response, err := client.Sync(context.Background(), "jwt", &since)

		require.NoError(t, err)
		require.Len(t, response.Secrets, 1)
		assert.Equal(t, "github", response.Secrets[0].Name)
	})

	t.Run("Должен вернуть API ошибку", func(t *testing.T) {
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(
					map[string]any{
						"error": map[string]string{"code": "unauthorized", "message": "authorization required"},
					},
				)
			}),
		)
		defer server.Close()
		client, err := NewClient(server.URL, server.Client())
		require.NoError(t, err)

		_, err = client.Sync(context.Background(), "bad-token", nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	t.Run("Должен вернуть ошибку без server URL", func(t *testing.T) {
		_, err := NewClient("", nil)

		require.Error(t, err)
	})
}
