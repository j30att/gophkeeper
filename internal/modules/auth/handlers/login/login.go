// Package login содержит HTTP-handler логина пользователя.
package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"github.com/igor/gophkeeper/internal/modules/auth/handlers/httputil"
	"github.com/igor/gophkeeper/internal/modules/auth/usecases"
)

// UseCase аутентифицирует пользователей.
type UseCase interface {
	Execute(ctx context.Context, input usecases.LoginInput) (usecases.LoginOutput, error)
}

// Handler обрабатывает HTTP-запросы логина пользователя.
type Handler struct {
	logger   zerolog.Logger
	useCase  UseCase
	validate *validator.Validate
}

type request struct {
	Login    string `json:"login" validate:"required,min=3"`
	Password string `json:"password" validate:"required,min=8"`
}

type response struct {
	Token string `json:"token"`
}

// New создает Handler.
func New(logger zerolog.Logger, useCase UseCase, validate *validator.Validate) (*Handler, error) {
	if useCase == nil {
		return nil, fmt.Errorf("%w: useCase", usecases.ErrEmptyDependency)
	}
	if validate == nil {
		return nil, fmt.Errorf("%w: validate", usecases.ErrEmptyDependency)
	}
	return &Handler{logger: logger, useCase: useCase, validate: validate}, nil
}

// ServeHTTP обрабатывает логин пользователя.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req request
	if !h.decodeAndValidate(w, r, &req) {
		return
	}

	output, err := h.useCase.Execute(
		r.Context(),
		usecases.LoginInput{Login: req.Login, Password: req.Password},
	)
	if err != nil {
		if errors.Is(err, usecases.ErrInvalidCredentials) {
			httputil.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials")
			return
		}
		h.logger.Error().Err(err).Msg("login failed")
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, response{Token: output.Token})
}

func (h *Handler) decodeAndValidate(w http.ResponseWriter, r *http.Request, req any) bool {
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return false
	}
	if err := h.validate.Struct(req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "validation_error", "request validation failed")
		return false
	}
	return true
}
