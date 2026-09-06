// Package delete содержит HTTP-handler удаления JSON-секрета.
package delete

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

// UseCase удаляет JSON-секрет.
type UseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) error
}

// Handler обрабатывает HTTP-запросы удаления JSON-секрета.
type Handler struct {
	logger  zerolog.Logger
	useCase UseCase
}

// New создает Handler.
func New(logger zerolog.Logger, useCase UseCase) (*Handler, error) {
	if useCase == nil {
		return nil, fmt.Errorf("%w: useCase", usecases.ErrEmptyDependency)
	}
	return &Handler{logger: logger, useCase: useCase}, nil
}

// ServeHTTP обрабатывает удаление JSON-секрета.
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

	if err = h.useCase.Execute(r.Context(), userID, secretID); err != nil {
		if httputil.WriteUseCaseError(w, err) {
			return
		}
		h.logger.Error().Err(err).Msg("delete secret failed")
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
