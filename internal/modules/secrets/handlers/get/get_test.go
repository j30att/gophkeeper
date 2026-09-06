package get

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
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
	output   usecases.SecretOutput
	err      error
	userID   uuid.UUID
	secretID uuid.UUID
}

func (m *useCaseMock) Execute(_ context.Context, userID uuid.UUID, secretID uuid.UUID) (usecases.SecretOutput, error) {
	m.userID = userID
	m.secretID = secretID
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
		"Должен вернуть secret", func(t *testing.T) {
			userID := uuid.New()
			secretID := uuid.New()
			useCase := &useCaseMock{
				output: usecases.SecretOutput{
					ID: secretID, UserID: userID, Type: domain.SecretTypeCredentials, Name: "github",
					Metadata: json.RawMessage(`{}`), Payload: json.RawMessage(`{}`), Version: 1,
				},
			}
			router := newTestRouter(t, useCase, userID)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/"+secretID.String(), nil)
			request.Header.Set("Authorization", "Bearer token")

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, userID, useCase.userID)
			assert.Equal(t, secretID, useCase.secretID)
		},
	)

	t.Run(
		"Должен вернуть 400 при невалидном id", func(t *testing.T) {
			router := newTestRouter(t, &useCaseMock{}, uuid.New())
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/bad", nil)
			request.Header.Set("Authorization", "Bearer token")

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
		},
	)

	t.Run(
		"Должен вернуть 404", func(t *testing.T) {
			router := newTestRouter(t, &useCaseMock{err: usecases.ErrSecretNotFound}, uuid.New())
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/"+uuid.New().String(), nil)
			request.Header.Set("Authorization", "Bearer token")

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusNotFound, recorder.Code)
		},
	)
}

func newTestRouter(t *testing.T, useCase UseCase, userID uuid.UUID) http.Handler {
	t.Helper()
	handler, err := New(zerolog.Nop(), useCase)
	require.NoError(t, err)
	router := chi.NewRouter()
	router.With(middleware.Auth(&tokenValidatorMock{claims: token.Claims{UserID: userID, Login: "igor"}})).
		Get("/api/v1/secrets/{id}", handler.ServeHTTP)
	return router
}
