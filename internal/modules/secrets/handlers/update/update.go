// Package update содержит HTTP-handler обновления JSON-секрета.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/igor/gophkeeper/internal/middleware"
	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
	"github.com/igor/gophkeeper/internal/modules/secrets/handlers/httputil"
	"github.com/igor/gophkeeper/internal/modules/secrets/usecases"
)

// UseCase обновляет JSON-секрет.
type UseCase interface {
	Execute(ctx context.Context, input usecases.UpdateSecretInput) (usecases.SecretOutput, error)
}

// Handler обрабатывает HTTP-запросы обновления JSON-секретов.
type Handler struct {
	logger   zerolog.Logger
	useCase  UseCase
	validate *validator.Validate
}

type request struct {
	Type     domain.SecretType `json:"type" validate:"required,oneof=credentials card"`
	Name     string            `json:"name" validate:"required,min=1"`
	Metadata json.RawMessage   `json:"metadata"`
	Payload  json.RawMessage   `json:"payload" validate:"required"`
}

type response struct {
	Secret usecases.SecretOutput `json:"secret"`
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

// ServeHTTP обрабатывает обновление JSON-секрета.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "authorization required")
		return
	}
	secretID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "invalid secret id")
		return
	}

	var req request
	if !h.decodeAndValidate(w, r, &req) {
		return
	}

	output, err := h.useCase.Execute(
		r.Context(),
		usecases.UpdateSecretInput{
			UserID:   userID,
			ID:       secretID,
			Type:     req.Type,
			Name:     req.Name,
			Metadata: req.Metadata,
			Payload:  req.Payload,
		},
	)
	if err != nil {
		if httputil.WriteUseCaseError(w, err) {
			return
		}
		h.logger.Error().Err(err).Msg("update secret failed")
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, response{Secret: output})
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
