// Package handlers содержит generated HTTP-handlers auth API.
package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/igor/gophkeeper/internal/modules/auth/usecases"
	"github.com/igor/gophkeeper/pkg/api/generated/auth"
)

// RegisterUseCase регистрирует пользователя.
type RegisterUseCase interface {
	Execute(ctx context.Context, input usecases.RegisterInput) (usecases.RegisterOutput, error)
}

// LoginUseCase аутентифицирует пользователя.
type LoginUseCase interface {
	Execute(ctx context.Context, input usecases.LoginInput) (usecases.LoginOutput, error)
}

// Handler реализует generated auth.StrictServerInterface.
type Handler struct {
	logger   zerolog.Logger
	register RegisterUseCase
	login    LoginUseCase
	validate *validator.Validate
}

// New создает Handler.
func New(
	logger zerolog.Logger,
	register RegisterUseCase,
	login LoginUseCase,
	validate *validator.Validate,
) (*Handler, error) {
	if register == nil {
		return nil, fmt.Errorf("%w: register", usecases.ErrEmptyDependency)
	}
	if login == nil {
		return nil, fmt.Errorf("%w: login", usecases.ErrEmptyDependency)
	}
	if validate == nil {
		return nil, fmt.Errorf("%w: validate", usecases.ErrEmptyDependency)
	}
	return &Handler{logger: logger, register: register, login: login, validate: validate}, nil
}

// PostApiV1AuthRegister регистрирует пользователя.
func (h *Handler) PostApiV1AuthRegister(
	ctx context.Context,
	request auth.PostApiV1AuthRegisterRequestObject,
) (auth.PostApiV1AuthRegisterResponseObject, error) {
	if request.Body == nil {
		return auth.PostApiV1AuthRegister400JSONResponse(errorResponse("bad_request", "invalid request body")), nil
	}
	if err := h.validate.Struct(request.Body); err != nil {
		return auth.PostApiV1AuthRegister400JSONResponse(errorResponse("validation_error", "request validation failed")), nil
	}

	output, err := h.register.Execute(
		ctx,
		usecases.RegisterInput{Login: request.Body.Login, Password: request.Body.Password},
	)
	if err != nil {
		if errors.Is(err, usecases.ErrUserAlreadyExists) {
			return auth.PostApiV1AuthRegister409JSONResponse(errorResponse("user_already_exists", "user already exists")), nil
		}
		h.logger.Error().Err(err).Msg("register failed")
		return auth.PostApiV1AuthRegister500JSONResponse(errorResponse("internal_error", "internal server error")), nil
	}

	return auth.PostApiV1AuthRegister201JSONResponse{
		UserId: uuid.UUID(output.UserID),
	}, nil
}

// PostApiV1AuthLogin аутентифицирует пользователя.
func (h *Handler) PostApiV1AuthLogin(
	ctx context.Context,
	request auth.PostApiV1AuthLoginRequestObject,
) (auth.PostApiV1AuthLoginResponseObject, error) {
	if request.Body == nil {
		return auth.PostApiV1AuthLogin400JSONResponse(errorResponse("bad_request", "invalid request body")), nil
	}
	if err := h.validate.Struct(request.Body); err != nil {
		return auth.PostApiV1AuthLogin400JSONResponse(errorResponse("validation_error", "request validation failed")), nil
	}

	output, err := h.login.Execute(
		ctx,
		usecases.LoginInput{Login: request.Body.Login, Password: request.Body.Password},
	)
	if err != nil {
		if errors.Is(err, usecases.ErrInvalidCredentials) {
			return auth.PostApiV1AuthLogin401JSONResponse(errorResponse("invalid_credentials", "invalid credentials")), nil
		}
		h.logger.Error().Err(err).Msg("login failed")
		return auth.PostApiV1AuthLogin500JSONResponse(errorResponse("internal_error", "internal server error")), nil
	}

	return auth.PostApiV1AuthLogin200JSONResponse{Token: output.Token}, nil
}

func errorResponse(code string, message string) auth.ErrorResponse {
	return auth.ErrorResponse{
		Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{Code: code, Message: message},
	}
}
