package update

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
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
	output usecases.SecretOutput
	err    error
	input  usecases.UpdateSecretInput
}

func (m *useCaseMock) Execute(_ context.Context, input usecases.UpdateSecretInput) (usecases.SecretOutput, error) {
	m.input = input
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
		"Должен обновить secret", func(t *testing.T) {
			userID := uuid.New()
			secretID := uuid.New()
			useCase := &useCaseMock{
				output: usecases.SecretOutput{
					ID: secretID, UserID: userID, Type: domain.SecretTypeCard, Name: "card",
					Metadata: json.RawMessage(`{}`), Payload: json.RawMessage(`{}`), Version: 2,
				},
			}
			router := newTestRouter(t, useCase, userID)
			recorder := httptest.NewRecorder()
			request := newJSONRequest(t, secretID.String(), request{
				Type: domain.SecretTypeCard, Name: "card", Metadata: json.RawMessage(`{}`),
				Payload: json.RawMessage(`{"number":"1234"}`),
			})

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, userID, useCase.input.UserID)
			assert.Equal(t, secretID, useCase.input.ID)
		},
	)

	t.Run(
		"Должен вернуть 400 при невалидном id", func(t *testing.T) {
			router := newTestRouter(t, &useCaseMock{}, uuid.New())
			recorder := httptest.NewRecorder()
			request := newJSONRequest(t, "bad", request{
				Type: domain.SecretTypeCard, Name: "card", Payload: json.RawMessage(`{}`),
			})

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
		},
	)

	t.Run(
		"Должен вернуть 400 при невалидном body", func(t *testing.T) {
			router := newTestRouter(t, &useCaseMock{}, uuid.New())
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPut, "/api/v1/secrets/"+uuid.New().String(), bytes.NewBufferString("{"))
			request.Header.Set("Authorization", "Bearer token")

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
		},
	)

	t.Run(
		"Должен вернуть 404", func(t *testing.T) {
			router := newTestRouter(t, &useCaseMock{err: usecases.ErrSecretNotFound}, uuid.New())
			recorder := httptest.NewRecorder()
			request := newJSONRequest(t, uuid.New().String(), request{
				Type: domain.SecretTypeCard, Name: "card", Payload: json.RawMessage(`{}`),
			})

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusNotFound, recorder.Code)
		},
	)
}

func newTestRouter(t *testing.T, useCase UseCase, userID uuid.UUID) http.Handler {
	t.Helper()
	handler, err := New(zerolog.Nop(), useCase, validator.New(validator.WithRequiredStructEnabled()))
	require.NoError(t, err)
	router := chi.NewRouter()
	router.With(middleware.Auth(&tokenValidatorMock{claims: token.Claims{UserID: userID, Login: "igor"}})).
		Put("/api/v1/secrets/{id}", handler.ServeHTTP)
	return router
}

func newJSONRequest(t *testing.T, id string, body any) *http.Request {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPut, "/api/v1/secrets/"+id, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer token")
	return request
}
