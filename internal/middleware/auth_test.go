package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/igor/gophkeeper/internal/modules/auth/token"
)

type tokenValidatorMock struct {
	claims      token.Claims
	err         error
	tokenString string
}

func (m *tokenValidatorMock) Validate(tokenString string) (token.Claims, error) {
	m.tokenString = tokenString
	return m.claims, m.err
}

func TestAuth(t *testing.T) {
	t.Run(
		"Должен пропустить запрос с валидным Bearer token", func(t *testing.T) {
			userID := uuid.New()
			validator := &tokenValidatorMock{
				claims: token.Claims{UserID: userID, Login: "igor"},
			}
			called := false
			next := http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					called = true
					actualUserID, ok := UserIDFromContext(r.Context())
					require.True(t, ok)
					assert.Equal(t, userID, actualUserID)
					login, ok := LoginFromContext(r.Context())
					require.True(t, ok)
					assert.Equal(t, "igor", login)
					w.WriteHeader(http.StatusNoContent)
				},
			)
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", "Bearer token-value")
			recorder := httptest.NewRecorder()

			Auth(validator)(next).ServeHTTP(recorder, request)

			require.Equal(t, http.StatusNoContent, recorder.Code)
			assert.True(t, called)
			assert.Equal(t, "token-value", validator.tokenString)
		},
	)

	t.Run(
		"Должен вернуть 500 без validator", func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", "Bearer token-value")
			recorder := httptest.NewRecorder()

			Auth(nil)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})).ServeHTTP(recorder, request)

			require.Equal(t, http.StatusInternalServerError, recorder.Code)
		},
	)

	t.Run(
		"Должен вернуть 401 без Authorization header", func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			recorder := httptest.NewRecorder()

			Auth(&tokenValidatorMock{})(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})).
				ServeHTTP(recorder, request)

			require.Equal(t, http.StatusUnauthorized, recorder.Code)
		},
	)

	t.Run(
		"Должен вернуть 401 при пустом Bearer token", func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", "Bearer ")
			recorder := httptest.NewRecorder()

			Auth(&tokenValidatorMock{})(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})).
				ServeHTTP(recorder, request)

			require.Equal(t, http.StatusUnauthorized, recorder.Code)
		},
	)

	t.Run(
		"Должен вернуть 401 при ошибке validation token", func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", "Bearer token-value")
			recorder := httptest.NewRecorder()

			Auth(&tokenValidatorMock{err: assert.AnError})(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {})).
				ServeHTTP(recorder, request)

			require.Equal(t, http.StatusUnauthorized, recorder.Code)
		},
	)
}

func TestContext(t *testing.T) {
	t.Run(
		"Должен вернуть false без user context", func(t *testing.T) {
			userID, ok := UserIDFromContext(context.Background())
			assert.False(t, ok)
			assert.Empty(t, userID)

			login, ok := LoginFromContext(context.Background())
			assert.False(t, ok)
			assert.Empty(t, login)
		},
	)
}
