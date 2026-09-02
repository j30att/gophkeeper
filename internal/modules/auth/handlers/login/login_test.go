package login

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/igor/gophkeeper/internal/modules/auth/usecases"
)

type useCaseMock struct {
	output usecases.LoginOutput
	err    error
	input  usecases.LoginInput
}

func (m *useCaseMock) Execute(_ context.Context, input usecases.LoginInput) (usecases.LoginOutput, error) {
	m.input = input
	return m.output, m.err
}

func TestHandler(t *testing.T) {
	t.Run(
		"Тест создания handler", func(t *testing.T) {
			t.Run(
				"Должен создать handler без ошибок", func(t *testing.T) {
					logger := zerolog.Nop()
					useCase := &useCaseMock{}
					validate := validator.New()

					handler, err := New(logger, useCase, validate)

					require.NoError(t, err)
					assert.Equal(t, useCase, handler.useCase)
					assert.Equal(t, validate, handler.validate)
				},
			)

			t.Run(
				"Ошибка, отсутствует зависимость", func(t *testing.T) {
					logger := zerolog.Nop()
					useCase := &useCaseMock{}
					validate := validator.New()

					tests := []struct {
						useCase  UseCase
						validate *validator.Validate
					}{
						{useCase: nil, validate: validate},
						{useCase: useCase, validate: nil},
					}

					for _, test := range tests {
						_, err := New(logger, test.useCase, test.validate)

						require.Error(t, err)
						assert.ErrorIs(t, err, usecases.ErrEmptyDependency)
					}
				},
			)
		},
	)

	t.Run(
		"Тест метода ServeHTTP", func(t *testing.T) {
			t.Run(
				"Должен вернуть 200", func(t *testing.T) {
					useCase := &useCaseMock{output: usecases.LoginOutput{Token: "jwt-token"}}
					handler := newTestHandler(t, useCase)
					recorder := httptest.NewRecorder()
					request := newJSONRequest(t, request{Login: "igor", Password: "password-1"})

					handler.ServeHTTP(recorder, request)

					require.Equal(t, http.StatusOK, recorder.Code)
					assert.Equal(t, "igor", useCase.input.Login)
					var response response
					require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
					assert.Equal(t, "jwt-token", response.Token)
				},
			)

			t.Run(
				"Должен вернуть 400 при невалидном JSON", func(t *testing.T) {
					handler := newTestHandler(t, &useCaseMock{})
					recorder := httptest.NewRecorder()
					request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString("{"))

					handler.ServeHTTP(recorder, request)

					require.Equal(t, http.StatusBadRequest, recorder.Code)
				},
			)

			t.Run(
				"Должен вернуть 400 при ошибке валидации", func(t *testing.T) {
					handler := newTestHandler(t, &useCaseMock{})
					recorder := httptest.NewRecorder()
					request := newJSONRequest(t, request{Login: "ig", Password: "short"})

					handler.ServeHTTP(recorder, request)

					require.Equal(t, http.StatusBadRequest, recorder.Code)
				},
			)

			t.Run(
				"Должен вернуть 401 при неверных credentials", func(t *testing.T) {
					handler := newTestHandler(t, &useCaseMock{err: usecases.ErrInvalidCredentials})
					recorder := httptest.NewRecorder()
					request := newJSONRequest(t, request{Login: "igor", Password: "password-1"})

					handler.ServeHTTP(recorder, request)

					require.Equal(t, http.StatusUnauthorized, recorder.Code)
				},
			)

			t.Run(
				"Должен вернуть 500 при ошибке useCase", func(t *testing.T) {
					handler := newTestHandler(t, &useCaseMock{err: assert.AnError})
					recorder := httptest.NewRecorder()
					request := newJSONRequest(t, request{Login: "igor", Password: "password-1"})

					handler.ServeHTTP(recorder, request)

					require.Equal(t, http.StatusInternalServerError, recorder.Code)
				},
			)
		},
	)
}

func newTestHandler(t *testing.T, useCase UseCase) *Handler {
	t.Helper()
	handler, err := New(zerolog.Nop(), useCase, validator.New(validator.WithRequiredStructEnabled()))
	require.NoError(t, err)
	return handler
}

func newJSONRequest(t *testing.T, body any) *http.Request {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	return request
}
