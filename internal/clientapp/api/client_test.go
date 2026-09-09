package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient(t *testing.T) {
	t.Run("Должен выполнить register", func(t *testing.T) {
		userID := uuid.NewString()
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/auth/register", r.URL.Path)
				assert.Empty(t, r.Header.Get("Authorization"))
				var request authRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				assert.Equal(t, "igor", request.Login)
				assert.Equal(t, "password-1", request.Password)
				_ = json.NewEncoder(w).Encode(map[string]string{"user_id": userID})
			}),
		)
		defer server.Close()
		client, err := NewClient(server.URL+"/", server.Client())
		require.NoError(t, err)

		actualUserID, err := client.Register(context.Background(), "igor", "password-1")

		require.NoError(t, err)
		assert.Equal(t, userID, actualUserID)
	})

	t.Run("Должен выполнить login", func(t *testing.T) {
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/auth/login", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
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

	t.Run("Должен получить secret", func(t *testing.T) {
		secretID := uuid.New()
		updatedAt := time.Date(2026, 9, 9, 1, 2, 3, 0, time.UTC)
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/v1/secrets/"+secretID.String(), r.URL.Path)
				assert.Equal(t, "Bearer jwt", r.Header.Get("Authorization"))
				_ = json.NewEncoder(w).Encode(
					map[string]Secret{
						"secret": {
							ID:        secretID,
							Name:      "github",
							Type:      SecretTypeCredentials,
							Metadata:  map[string]interface{}{"site": "github.com"},
							Payload:   map[string]interface{}{"login": "igor"},
							Version:   2,
							UpdatedAt: updatedAt,
						},
					},
				)
			}),
		)
		defer server.Close()
		client, err := NewClient(server.URL, server.Client())
		require.NoError(t, err)

		secret, err := client.GetSecret(context.Background(), "jwt", secretID)

		require.NoError(t, err)
		assert.Equal(t, secretID, secret.ID)
		assert.Equal(t, "github", secret.Name)
		assert.Equal(t, 2, secret.Version)
	})

	t.Run("Должен создать secret", func(t *testing.T) {
		secretID := uuid.New()
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/secrets", r.URL.Path)
				assert.Equal(t, "Bearer jwt", r.Header.Get("Authorization"))
				var input SecretInput
				require.NoError(t, json.NewDecoder(r.Body).Decode(&input))
				assert.Equal(t, SecretTypeCredentials, input.Type)
				assert.Equal(t, "github", input.Name)
				_ = json.NewEncoder(w).Encode(
					map[string]Secret{
						"secret": {
							ID:      secretID,
							Name:    input.Name,
							Type:    input.Type,
							Payload: input.Payload,
						},
					},
				)
			}),
		)
		defer server.Close()
		client, err := NewClient(server.URL, server.Client())
		require.NoError(t, err)

		secret, err := client.CreateSecret(
			context.Background(),
			"jwt",
			SecretInput{
				Type:    SecretTypeCredentials,
				Name:    "github",
				Payload: map[string]interface{}{"login": "igor"},
			},
		)

		require.NoError(t, err)
		assert.Equal(t, secretID, secret.ID)
		assert.Equal(t, "github", secret.Name)
	})

	t.Run("Должен обновить secret", func(t *testing.T) {
		secretID := uuid.New()
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, "/api/v1/secrets/"+secretID.String(), r.URL.Path)
				assert.Equal(t, "Bearer jwt", r.Header.Get("Authorization"))
				var input SecretInput
				require.NoError(t, json.NewDecoder(r.Body).Decode(&input))
				assert.Equal(t, SecretTypeCredentials, input.Type)
				assert.Equal(t, "github", input.Name)
				assert.Equal(t, 2, input.ExpectedVersion)
				_ = json.NewEncoder(w).Encode(
					map[string]Secret{
						"secret": {
							ID:      secretID,
							Name:    input.Name,
							Type:    input.Type,
							Payload: input.Payload,
							Version: 3,
						},
					},
				)
			}),
		)
		defer server.Close()
		client, err := NewClient(server.URL, server.Client())
		require.NoError(t, err)

		secret, err := client.UpdateSecret(
			context.Background(),
			"jwt",
			secretID,
			SecretInput{
				Type:            SecretTypeCredentials,
				Name:            "github",
				Payload:         map[string]interface{}{"login": "igor"},
				ExpectedVersion: 2,
			},
		)

		require.NoError(t, err)
		assert.Equal(t, 3, secret.Version)
		assert.Equal(t, "github", secret.Name)
	})

	t.Run("Должен удалить secret", func(t *testing.T) {
		secretID := uuid.New()
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/api/v1/secrets/"+secretID.String(), r.URL.Path)
				assert.Equal(t, "Bearer jwt", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusNoContent)
			}),
		)
		defer server.Close()
		client, err := NewClient(server.URL, server.Client())
		require.NoError(t, err)

		err = client.DeleteSecret(context.Background(), "jwt", secretID)

		require.NoError(t, err)
	})

	t.Run("Должен создать blob secret", func(t *testing.T) {
		secretID := uuid.New()
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/secrets/blob", r.URL.Path)
				assert.Equal(t, "Bearer jwt", r.Header.Get("Authorization"))
				require.NoError(t, r.ParseMultipartForm(1024))
				assert.Equal(t, "text", r.FormValue("type"))
				assert.Equal(t, "note", r.FormValue("name"))
				assert.JSONEq(t, `{"kind":"book"}`, r.FormValue("metadata"))
				file, header, err := r.FormFile("file")
				require.NoError(t, err)
				defer file.Close()
				content, err := io.ReadAll(file)
				require.NoError(t, err)
				assert.Equal(t, "note.txt", header.Filename)
				assert.Equal(t, "content", string(content))
				_ = json.NewEncoder(w).Encode(
					map[string]Secret{
						"secret": {
							ID:   secretID,
							Name: "note",
							Type: SecretTypeText,
							Blob: &Blob{OriginalName: "note.txt", Size: int64(len(content))},
						},
					},
				)
			}),
		)
		defer server.Close()
		client, err := NewClient(server.URL, server.Client())
		require.NoError(t, err)

		secret, err := client.CreateBlobSecret(
			context.Background(),
			"jwt",
			BlobSecretInput{
				Type:         SecretTypeText,
				Name:         "note",
				Metadata:     `{"kind":"book"}`,
				OriginalName: "note.txt",
				ContentType:  "text/plain",
				Content:      strings.NewReader("content"),
			},
		)

		require.NoError(t, err)
		assert.Equal(t, secretID, secret.ID)
		require.NotNil(t, secret.Blob)
		assert.Equal(t, "note.txt", secret.Blob.OriginalName)
	})

	t.Run("Должен скачать blob content", func(t *testing.T) {
		secretID := uuid.New()
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/v1/secrets/"+secretID.String()+"/content", r.URL.Path)
				assert.Equal(t, "Bearer jwt", r.Header.Get("Authorization"))
				_, _ = w.Write([]byte("content"))
			}),
		)
		defer server.Close()
		client, err := NewClient(server.URL, server.Client())
		require.NoError(t, err)
		var output strings.Builder

		err = client.DownloadBlobContent(context.Background(), "jwt", secretID, &output)

		require.NoError(t, err)
		assert.Equal(t, "content", output.String())
	})

	t.Run("Должен заменить blob content", func(t *testing.T) {
		secretID := uuid.New()
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPut, r.Method)
				assert.Equal(t, "/api/v1/secrets/"+secretID.String()+"/content", r.URL.Path)
				assert.Equal(t, "Bearer jwt", r.Header.Get("Authorization"))
				require.NoError(t, r.ParseMultipartForm(1024))
				assert.Equal(t, "2", r.FormValue("expected_version"))
				file, header, err := r.FormFile("file")
				require.NoError(t, err)
				defer file.Close()
				content, err := io.ReadAll(file)
				require.NoError(t, err)
				assert.Equal(t, "new.txt", header.Filename)
				assert.Equal(t, "new content", string(content))
				_ = json.NewEncoder(w).Encode(
					map[string]Secret{
						"secret": {
							ID:      secretID,
							Name:    "note",
							Type:    SecretTypeText,
							Version: 3,
							Blob:    &Blob{OriginalName: "new.txt", Size: int64(len(content))},
						},
					},
				)
			}),
		)
		defer server.Close()
		client, err := NewClient(server.URL, server.Client())
		require.NoError(t, err)

		secret, err := client.UpdateBlobContent(
			context.Background(),
			"jwt",
			secretID,
			BlobContentInput{
				ExpectedVersion: 2,
				OriginalName:    "new.txt",
				ContentType:     "text/plain",
				Content:         strings.NewReader("new content"),
			},
		)

		require.NoError(t, err)
		assert.Equal(t, 3, secret.Version)
		require.NotNil(t, secret.Blob)
		assert.Equal(t, "new.txt", secret.Blob.OriginalName)
	})

	t.Run("Должен вернуть ошибку blob upload без content", func(t *testing.T) {
		client, err := NewClient("http://127.0.0.1", nil)
		require.NoError(t, err)

		_, err = client.CreateBlobSecret(context.Background(), "jwt", BlobSecretInput{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty file content")
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

	t.Run("Должен вернуть status при нечитаемой API ошибке", func(t *testing.T) {
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("{"))
			}),
		)
		defer server.Close()
		client, err := NewClient(server.URL, server.Client())
		require.NoError(t, err)

		_, err = client.Login(context.Background(), "igor", "password-1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 500")
	})

	t.Run("Должен вернуть status при API ошибке без message", func(t *testing.T) {
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "bad_request"}})
			}),
		)
		defer server.Close()
		client, err := NewClient(server.URL, server.Client())
		require.NoError(t, err)

		_, err = client.Login(context.Background(), "igor", "password-1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 400")
	})

	t.Run("Должен вернуть ошибку decode response", func(t *testing.T) {
		server := httptest.NewServer(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("{"))
			}),
		)
		defer server.Close()
		client, err := NewClient(server.URL, server.Client())
		require.NoError(t, err)

		_, err = client.Login(context.Background(), "igor", "password-1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode response")
	})

	t.Run("Должен вернуть ошибку marshal request", func(t *testing.T) {
		client, err := NewClient("http://127.0.0.1", nil)
		require.NoError(t, err)

		err = client.doJSON(context.Background(), http.MethodPost, "/test", "", map[string]interface{}{"bad": func() {}}, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "marshal request")
	})

	t.Run("Должен вернуть ошибку create request", func(t *testing.T) {
		client, err := NewClient("http://[::1", nil)
		require.NoError(t, err)

		err = client.doJSON(context.Background(), http.MethodGet, "/test", "", nil, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "create request")
	})

	t.Run("Должен вернуть ошибку без server URL", func(t *testing.T) {
		_, err := NewClient("", nil)

		require.Error(t, err)
	})

	t.Run("Должен вернуть ошибку только из пробелов вместо server URL", func(t *testing.T) {
		_, err := NewClient("   ", nil)

		require.Error(t, err)
	})
}
