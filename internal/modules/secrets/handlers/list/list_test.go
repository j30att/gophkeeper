package list

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/igor/gophkeeper/internal/middleware"
	"github.com/igor/gophkeeper/internal/modules/auth/token"
	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
	"github.com/igor/gophkeeper/internal/modules/secrets/usecases"
)

type useCaseMock struct {
	output []usecases.SecretListItem
	err    error
	userID uuid.UUID
}

func (m *useCaseMock) Execute(_ context.Context, userID uuid.UUID) ([]usecases.SecretListItem, error) {
	m.userID = userID
	return m.output, m.err
}

type tokenValidatorMock struct {
	claims token.Claims
}

func (m *tokenValidatorMock) Validate(_ string) (token.Claims, error) {
	return m.claims, nil
}

func TestHandler(t *testing.T) {
	t.Run(
		"Должен вернуть список", func(t *testing.T) {
			userID := uuid.New()
			useCase := &useCaseMock{
				output: []usecases.SecretListItem{
					{ID: uuid.New(), Type: domain.SecretTypeCredentials, Name: "github", Metadata: json.RawMessage(`{}`), Version: 1},
				},
			}
			handler := newTestHandler(t, useCase, userID)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/secrets", nil)
			request.Header.Set("Authorization", "Bearer token")

			handler.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, userID, useCase.userID)
		},
	)

	t.Run(
		"Должен вернуть 500 при ошибке usecase", func(t *testing.T) {
			handler := newTestHandler(t, &useCaseMock{err: assert.AnError}, uuid.New())
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/secrets", nil)
			request.Header.Set("Authorization", "Bearer token")

			handler.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusInternalServerError, recorder.Code)
		},
	)
}

func newTestHandler(t *testing.T, useCase UseCase, userID uuid.UUID) http.Handler {
	t.Helper()
	handler, err := New(zerolog.Nop(), useCase)
	require.NoError(t, err)
	return middleware.Auth(&tokenValidatorMock{claims: token.Claims{UserID: userID, Login: "igor"}})(handler)
}
