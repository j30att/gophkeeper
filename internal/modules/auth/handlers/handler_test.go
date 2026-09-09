package handlers

import (
	"context"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/igor/gophkeeper/internal/modules/auth/usecases"
	"github.com/igor/gophkeeper/pkg/api/generated/auth"
)

type registerUseCaseMock struct {
	output usecases.RegisterOutput
	err    error
	input  usecases.RegisterInput
}

func (m *registerUseCaseMock) Execute(_ context.Context, input usecases.RegisterInput) (usecases.RegisterOutput, error) {
	m.input = input
	return m.output, m.err
}

type loginUseCaseMock struct {
	output usecases.LoginOutput
	err    error
	input  usecases.LoginInput
}

func (m *loginUseCaseMock) Execute(_ context.Context, input usecases.LoginInput) (usecases.LoginOutput, error) {
	m.input = input
	return m.output, m.err
}

func TestHandlerNew(t *testing.T) {
	t.Run(
		"Должен создать handler", func(t *testing.T) {
			registerUseCase := &registerUseCaseMock{}
			loginUseCase := &loginUseCaseMock{}
			validate := validator.New()

			handler, err := New(zerolog.Nop(), registerUseCase, loginUseCase, validate)

			require.NoError(t, err)
			assert.Equal(t, registerUseCase, handler.register)
			assert.Equal(t, loginUseCase, handler.login)
			assert.Equal(t, validate, handler.validate)
		},
	)

	t.Run(
		"Должен вернуть ошибку без зависимостей", func(t *testing.T) {
			registerUseCase := &registerUseCaseMock{}
			loginUseCase := &loginUseCaseMock{}
			validate := validator.New()

			tests := []struct {
				register RegisterUseCase
				login    LoginUseCase
				validate *validator.Validate
			}{
				{register: nil, login: loginUseCase, validate: validate},
				{register: registerUseCase, login: nil, validate: validate},
				{register: registerUseCase, login: loginUseCase, validate: nil},
			}

			for _, test := range tests {
				_, err := New(zerolog.Nop(), test.register, test.login, test.validate)

				require.Error(t, err)
				assert.ErrorIs(t, err, usecases.ErrEmptyDependency)
			}
		},
	)
}

func TestHandlerPostApiV1AuthRegister(t *testing.T) {
	t.Run(
		"Должен вернуть 201", func(t *testing.T) {
			userID := uuid.New()
			registerUseCase := &registerUseCaseMock{output: usecases.RegisterOutput{UserID: userID}}
			handler := newTestHandler(t, registerUseCase, &loginUseCaseMock{})

			response, err := handler.PostApiV1AuthRegister(
				context.Background(),
				auth.PostApiV1AuthRegisterRequestObject{Body: &auth.AuthRequest{Login: "igor", Password: "password-1"}},
			)

			require.NoError(t, err)
			created, ok := response.(auth.PostApiV1AuthRegister201JSONResponse)
			require.True(t, ok)
			assert.Equal(t, userID, uuid.UUID(created.UserId))
			assert.Equal(t, usecases.RegisterInput{Login: "igor", Password: "password-1"}, registerUseCase.input)
		},
	)

	t.Run(
		"Должен вернуть 400 при ошибке валидации", func(t *testing.T) {
			handler := newTestHandler(t, &registerUseCaseMock{}, &loginUseCaseMock{})

			response, err := handler.PostApiV1AuthRegister(
				context.Background(),
				auth.PostApiV1AuthRegisterRequestObject{Body: &auth.AuthRequest{Login: "ig", Password: "short"}},
			)

			require.NoError(t, err)
			_, ok := response.(auth.PostApiV1AuthRegister400JSONResponse)
			assert.True(t, ok)
		},
	)

	t.Run(
		"Должен вернуть 409 при существующем пользователе", func(t *testing.T) {
			handler := newTestHandler(t, &registerUseCaseMock{err: usecases.ErrUserAlreadyExists}, &loginUseCaseMock{})

			response, err := handler.PostApiV1AuthRegister(
				context.Background(),
				auth.PostApiV1AuthRegisterRequestObject{Body: &auth.AuthRequest{Login: "igor", Password: "password-1"}},
			)

			require.NoError(t, err)
			_, ok := response.(auth.PostApiV1AuthRegister409JSONResponse)
			assert.True(t, ok)
		},
	)
}

func TestHandlerPostApiV1AuthLogin(t *testing.T) {
	t.Run(
		"Должен вернуть 200", func(t *testing.T) {
			loginUseCase := &loginUseCaseMock{output: usecases.LoginOutput{Token: "token"}}
			handler := newTestHandler(t, &registerUseCaseMock{}, loginUseCase)

			response, err := handler.PostApiV1AuthLogin(
				context.Background(),
				auth.PostApiV1AuthLoginRequestObject{Body: &auth.AuthRequest{Login: "igor", Password: "password-1"}},
			)

			require.NoError(t, err)
			okResponse, ok := response.(auth.PostApiV1AuthLogin200JSONResponse)
			require.True(t, ok)
			assert.Equal(t, "token", okResponse.Token)
			assert.Equal(t, usecases.LoginInput{Login: "igor", Password: "password-1"}, loginUseCase.input)
		},
	)

	t.Run(
		"Должен вернуть 401 при неверных credentials", func(t *testing.T) {
			handler := newTestHandler(t, &registerUseCaseMock{}, &loginUseCaseMock{err: usecases.ErrInvalidCredentials})

			response, err := handler.PostApiV1AuthLogin(
				context.Background(),
				auth.PostApiV1AuthLoginRequestObject{Body: &auth.AuthRequest{Login: "igor", Password: "password-1"}},
			)

			require.NoError(t, err)
			_, ok := response.(auth.PostApiV1AuthLogin401JSONResponse)
			assert.True(t, ok)
		},
	)
}

func newTestHandler(t *testing.T, registerUseCase RegisterUseCase, loginUseCase LoginUseCase) *Handler {
	t.Helper()
	handler, err := New(
		zerolog.Nop(),
		registerUseCase,
		loginUseCase,
		validator.New(validator.WithRequiredStructEnabled()),
	)
	require.NoError(t, err)
	return handler
}
