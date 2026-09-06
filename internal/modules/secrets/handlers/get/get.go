// Package get содержит HTTP-handler получения JSON-секрета.
package get

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/igor/gophkeeper/internal/middleware"
	"github.com/igor/gophkeeper/internal/modules/secrets/handlers/httputil"
	"github.com/igor/gophkeeper/internal/modules/secrets/usecases"
)

// UseCase возвращает JSON-секрет.
type UseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (usecases.SecretOutput, error)
}

// Handler обрабатывает HTTP-запросы получения JSON-секрета.
type Handler struct {
	logger  zerolog.Logger
	useCase UseCase
}

type response struct {
	Secret usecases.SecretOutput `json:"secret"`
}

// New создает Handler.
func New(logger zerolog.Logger, useCase UseCase) (*Handler, error) {
	if useCase == nil {
		return nil, fmt.Errorf("%w: useCase", usecases.ErrEmptyDependency)
	}
	return &Handler{logger: logger, useCase: useCase}, nil
}

// ServeHTTP обрабатывает получение JSON-секрета.
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

	output, err := h.useCase.Execute(r.Context(), userID, secretID)
	if err != nil {
		if httputil.WriteUseCaseError(w, err) {
			return
		}
		h.logger.Error().Err(err).Msg("get secret failed")
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, response{Secret: output})
}
