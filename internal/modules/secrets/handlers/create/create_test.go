package create

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
	input  usecases.SecretInput
}

func (m *useCaseMock) Execute(_ context.Context, input usecases.SecretInput) (usecases.SecretOutput, error) {
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
		"Должен создать handler", func(t *testing.T) {
			handler, err := New(zerolog.Nop(), &useCaseMock{}, validator.New())

			require.NoError(t, err)
			assert.NotNil(t, handler)
		},
	)

	t.Run(
		"Должен вернуть ошибку без зависимостей", func(t *testing.T) {
			_, err := New(zerolog.Nop(), nil, validator.New())
			require.Error(t, err)
			assert.ErrorIs(t, err, usecases.ErrEmptyDependency)

			_, err = New(zerolog.Nop(), &useCaseMock{}, nil)
			require.Error(t, err)
			assert.ErrorIs(t, err, usecases.ErrEmptyDependency)
		},
	)

	t.Run(
		"Должен вернуть 201", func(t *testing.T) {
			userID := uuid.New()
			secretID := uuid.New()
			useCase := &useCaseMock{
				output: usecases.SecretOutput{
					ID: userID, UserID: userID, Type: domain.SecretTypeCredentials, Name: "github",
					Metadata: json.RawMessage(`{}`), Payload: json.RawMessage(`{"login":"igor"}`), Version: 1,
				},
			}
			handler := newTestHandler(t, useCase, userID)
			recorder := httptest.NewRecorder()
			request := newJSONRequest(t, request{
				Type: domain.SecretTypeCredentials, Name: "github", Metadata: json.RawMessage(`{}`),
				Payload: json.RawMessage(`{"login":"igor"}`),
			})
			useCase.output.ID = secretID

			handler.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusCreated, recorder.Code)
			assert.Equal(t, userID, useCase.input.UserID)
			assert.Equal(t, "github", useCase.input.Name)
		},
	)

	t.Run(
		"Должен вернуть 400 при невалидном body", func(t *testing.T) {
			handler := newTestHandler(t, &useCaseMock{}, uuid.New())
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewBufferString("{"))
			request.Header.Set("Authorization", "Bearer token")

			handler.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
		},
	)

	t.Run(
		"Должен вернуть 400 при ошибке usecase validation", func(t *testing.T) {
			handler := newTestHandler(t, &useCaseMock{err: usecases.ErrInvalidSecretType}, uuid.New())
			recorder := httptest.NewRecorder()
			request := newJSONRequest(t, request{
				Type: domain.SecretTypeCredentials, Name: "github", Payload: json.RawMessage(`{}`),
			})

			handler.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
		},
	)
}

func newTestHandler(t *testing.T, useCase UseCase, userID uuid.UUID) http.Handler {
	t.Helper()
	handler, err := New(zerolog.Nop(), useCase, validator.New(validator.WithRequiredStructEnabled()))
	require.NoError(t, err)
	return middleware.Auth(&tokenValidatorMock{claims: token.Claims{UserID: userID, Login: "igor"}})(handler)
}

func newJSONRequest(t *testing.T, body any) *http.Request {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/secrets", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer token")
	return request
}
