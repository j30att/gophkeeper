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

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/igor/gophkeeper/internal/config"
)

func TestBuildRouter(t *testing.T) {
	t.Run(
		"Должен создать router и вернуть health check", func(t *testing.T) {
			router, err := BuildInMemoryRouter(zerolog.Nop())
			require.NoError(t, err)

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.JSONEq(t, `{"status":"ok"}`, recorder.Body.String())
		},
	)

	t.Run(
		"Должен зарегистрировать пользователя и выполнить login", func(t *testing.T) {
			router, err := BuildInMemoryRouter(zerolog.Nop())
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
		"Должен вернуть ошибку без user repository", func(t *testing.T) {
			_, err := BuildRouter(zerolog.Nop(), testConfig(), nil)

			require.Error(t, err)
		},
	)
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

func testConfig() config.Config {
	return config.Config{
		Server: config.Server{Address: ":8080"},
		Auth: config.Auth{
			JWTSecret:      "secret",
			AccessTokenTTL: time.Hour,
		},
	}
}
