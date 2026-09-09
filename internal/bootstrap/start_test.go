package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/igor/gophkeeper/internal/config"
	authdomain "github.com/igor/gophkeeper/internal/modules/auth/domain"
	authusecases "github.com/igor/gophkeeper/internal/modules/auth/usecases"
	secretsdomain "github.com/igor/gophkeeper/internal/modules/secrets/domain"
	secretsusecases "github.com/igor/gophkeeper/internal/modules/secrets/usecases"
)

func TestBuildRouter(t *testing.T) {
	t.Run(
		"Должен создать router и вернуть health check", func(t *testing.T) {
			router, err := BuildRouter(zerolog.Nop(), testConfig(t), newTestUserRepository(), testSecretsRepository{})
			require.NoError(t, err)

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.JSONEq(t, `{"status":"ok"}`, recorder.Body.String())
		},
	)

	t.Run(
		"Должен отдать OpenAPI spec", func(t *testing.T) {
			router, err := BuildRouter(zerolog.Nop(), testConfig(t), newTestUserRepository(), testSecretsRepository{})
			require.NoError(t, err)

			request := httptest.NewRequest(http.MethodGet, "/openapi.yml", nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, "application/yaml", recorder.Header().Get("Content-Type"))
			assert.Contains(t, recorder.Body.String(), "openapi: 3.0.3")
		},
	)

	t.Run(
		"Должен отдать Swagger UI", func(t *testing.T) {
			router, err := BuildRouter(zerolog.Nop(), testConfig(t), newTestUserRepository(), testSecretsRepository{})
			require.NoError(t, err)

			request := httptest.NewRequest(http.MethodGet, "/docs", nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, "text/html", recorder.Header().Get("Content-Type"))
			assert.Contains(t, recorder.Body.String(), "SwaggerUIBundle")
			assert.Contains(t, recorder.Body.String(), `url: "/openapi.yml"`)
		},
	)

	t.Run(
		"Должен зарегистрировать пользователя и выполнить login", func(t *testing.T) {
			router, err := BuildRouter(zerolog.Nop(), testConfig(t), newTestUserRepository(), testSecretsRepository{})
			require.NoError(t, err)

			registerRecorder := httptest.NewRecorder()
			registerRequest := newBootstrapJSONRequest(
				t,
				"/api/v1/auth/register",
				map[string]string{"login": "igor", "password": "password-1"},
			)
			router.ServeHTTP(registerRecorder, registerRequest)
			require.Equal(t, http.StatusCreated, registerRecorder.Code)

			loginRecorder := httptest.NewRecorder()
			loginRequest := newBootstrapJSONRequest(
				t,
				"/api/v1/auth/login",
				map[string]string{"login": "igor", "password": "password-1"},
			)
			router.ServeHTTP(loginRecorder, loginRequest)

			require.Equal(t, http.StatusOK, loginRecorder.Code)
			var response struct {
				Token string `json:"token"`
			}
			require.NoError(t, json.NewDecoder(loginRecorder.Body).Decode(&response))
			assert.NotEmpty(t, response.Token)
		},
	)

	t.Run(
		"Должен вернуть JSON-ошибку при невалидном request body", func(t *testing.T) {
			router, err := BuildRouter(zerolog.Nop(), testConfig(t), newTestUserRepository(), testSecretsRepository{})
			require.NoError(t, err)

			request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("{"))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
			assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
			assert.JSONEq(
				t,
				`{"error":{"code":"bad_request","message":"invalid request"}}`,
				recorder.Body.String(),
			)
		},
	)

	t.Run(
		"Должен вернуть ошибку без user repository", func(t *testing.T) {
			_, err := BuildRouter(zerolog.Nop(), testConfig(t), nil, testSecretsRepository{})

			require.Error(t, err)
		},
	)

	t.Run(
		"Должен вернуть ошибку без secrets repository", func(t *testing.T) {
			_, err := BuildRouter(zerolog.Nop(), testConfig(t), newTestUserRepository(), nil)

			require.Error(t, err)
		},
	)
}

type testUserRepository struct {
	byLogin map[string]authdomain.User
}

func newTestUserRepository() *testUserRepository {
	return &testUserRepository{byLogin: make(map[string]authdomain.User)}
}

func (r *testUserRepository) Save(_ context.Context, user authdomain.User) error {
	r.byLogin[user.Login] = user
	return nil
}

func (r *testUserRepository) Load(_ context.Context, login string) (authdomain.User, error) {
	user, ok := r.byLogin[login]
	if !ok {
		return authdomain.User{}, authusecases.ErrUserNotFound
	}
	return user, nil
}

type testSecretsRepository struct{}

func (testSecretsRepository) Save(_ context.Context, _ secretsdomain.Secret) error {
	return nil
}

func (testSecretsRepository) Load(_ context.Context, _ uuid.UUID, _ uuid.UUID) (secretsdomain.Secret, error) {
	return secretsdomain.Secret{}, secretsusecases.ErrSecretNotFound
}

func (testSecretsRepository) List(_ context.Context, _ uuid.UUID) ([]secretsdomain.Secret, error) {
	return nil, nil
}

func (testSecretsRepository) ListChanged(_ context.Context, _ uuid.UUID, _ time.Time) ([]secretsdomain.Secret, error) {
	return nil, nil
}

func (testSecretsRepository) Update(_ context.Context, _ secretsdomain.Secret, _ int) error {
	return secretsusecases.ErrSecretNotFound
}

func (testSecretsRepository) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return secretsusecases.ErrSecretNotFound
}

func (testSecretsRepository) SaveBlob(_ context.Context, _ secretsdomain.Blob) error {
	return nil
}

func (testSecretsRepository) LoadBlobBySecret(
	_ context.Context,
	_ uuid.UUID,
	_ uuid.UUID,
) (secretsdomain.Blob, error) {
	return secretsdomain.Blob{}, secretsusecases.ErrBlobNotFound
}

func (testSecretsRepository) MarkBlobDeleted(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return secretsusecases.ErrBlobNotFound
}

func (testSecretsRepository) ListBlobsForCleanup(
	_ context.Context,
	_ time.Time,
	_ int,
) ([]secretsdomain.Blob, error) {
	return nil, nil
}

func (testSecretsRepository) MarkBlobStorageDeleted(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return secretsusecases.ErrBlobNotFound
}

func TestRunHTTPServer(t *testing.T) {
	t.Run(
		"Должен вернуть nil при ErrServerClosed", func(t *testing.T) {
			err := runHTTPServer(
				context.Background(),
				zerolog.Nop(),
				func() error { return http.ErrServerClosed },
				func(_ context.Context) error { return nil },
			)

			require.NoError(t, err)
		},
	)

	t.Run(
		"Должен вернуть ошибку listenAndServe", func(t *testing.T) {
			err := runHTTPServer(
				context.Background(),
				zerolog.Nop(),
				func() error { return assert.AnError },
				func(_ context.Context) error { return nil },
			)

			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
		},
	)

	t.Run(
		"Должен выполнить shutdown при отмене context", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			shutdownCalled := false

			err := runHTTPServer(
				ctx,
				zerolog.Nop(),
				func() error {
					select {}
				},
				func(_ context.Context) error {
					shutdownCalled = true
					return nil
				},
			)

			require.NoError(t, err)
			assert.True(t, shutdownCalled)
		},
	)

	t.Run(
		"Должен вернуть ошибку shutdown", func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			err := runHTTPServer(
				ctx,
				zerolog.Nop(),
				func() error {
					select {}
				},
				func(_ context.Context) error { return assert.AnError },
			)

			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
		},
	)

	t.Run(
		"Должен сохранить wrapping ошибки listenAndServe", func(t *testing.T) {
			sentinel := errors.New("listen failed")

			err := runHTTPServer(
				context.Background(),
				zerolog.Nop(),
				func() error { return sentinel },
				func(_ context.Context) error { return nil },
			)

			require.Error(t, err)
			assert.ErrorIs(t, err, sentinel)
		},
	)
}

func newBootstrapJSONRequest(t *testing.T, path string, body any) *http.Request {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func testConfig(t *testing.T) config.Config {
	t.Helper()
	return config.Config{
		Server: config.Server{Address: ":8080"},
		Auth: config.Auth{
			JWTSecret:      "secret",
			AccessTokenTTL: time.Hour,
		},
		Crypto: config.Crypto{MasterKey: "secret"},
		Storage: config.Storage{
			BlobPath: t.TempDir(),
		},
	}
}
