// Package list содержит HTTP-handler списка JSON-секретов.
package list

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/igor/gophkeeper/internal/middleware"
	"github.com/igor/gophkeeper/internal/modules/secrets/handlers/httputil"
	"github.com/igor/gophkeeper/internal/modules/secrets/usecases"
)

// UseCase возвращает список JSON-секретов.
type UseCase interface {
	Execute(ctx context.Context, userID uuid.UUID) ([]usecases.SecretListItem, error)
}

// Handler обрабатывает HTTP-запросы списка JSON-секретов.
type Handler struct {
	logger  zerolog.Logger
	useCase UseCase
}

type response struct {
	Secrets []usecases.SecretListItem `json:"secrets"`
}

// New создает Handler.
func New(logger zerolog.Logger, useCase UseCase) (*Handler, error) {
	if useCase == nil {
		return nil, fmt.Errorf("%w: useCase", usecases.ErrEmptyDependency)
	}
	return &Handler{logger: logger, useCase: useCase}, nil
}

// ServeHTTP обрабатывает получение списка JSON-секретов.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "authorization required")
		return
	}

	output, err := h.useCase.Execute(r.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Msg("list secrets failed")
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	httputil.WriteJSON(w, http.StatusOK, response{Secrets: output})
}
