// Package register содержит HTTP-handler регистрации пользователя.
package register

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/igor/gophkeeper/internal/modules/auth/handlers/httputil"
	"github.com/igor/gophkeeper/internal/modules/auth/usecases"
)

// UseCase регистрирует пользователей.
type UseCase interface {
	Execute(ctx context.Context, input usecases.RegisterInput) (usecases.RegisterOutput, error)
}

// Handler обрабатывает HTTP-запросы регистрации пользователя.
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
	UserID uuid.UUID `json:"user_id"`
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

// ServeHTTP обрабатывает регистрацию пользователя.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req request
	if !h.decodeAndValidate(w, r, &req) {
		return
	}

	output, err := h.useCase.Execute(
		r.Context(),
		usecases.RegisterInput{Login: req.Login, Password: req.Password},
	)
	if err != nil {
		if errors.Is(err, usecases.ErrUserAlreadyExists) {
			httputil.WriteError(w, http.StatusConflict, "user_already_exists", "user already exists")
			return
		}
		h.logger.Error().Err(err).Msg("register failed")
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, response{UserID: output.UserID})
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
