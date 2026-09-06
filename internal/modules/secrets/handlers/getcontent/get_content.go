// Package getcontent содержит HTTP-handler чтения содержимого blob-секрета.
package getcontent

import (
	"context"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/igor/gophkeeper/internal/middleware"
	"github.com/igor/gophkeeper/internal/modules/secrets/handlers/httputil"
	"github.com/igor/gophkeeper/internal/modules/secrets/usecases"
)

// UseCase возвращает stream содержимого blob-секрета.
type UseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (usecases.BlobContentOutput, error)
}

// Handler обрабатывает запрос чтения blob-содержимого.
type Handler struct {
	logger  zerolog.Logger
	useCase UseCase
}

// New создает Handler.
func New(logger zerolog.Logger, useCase UseCase) (*Handler, error) {
	if useCase == nil {
		return nil, usecases.ErrEmptyDependency
	}
	return &Handler{logger: logger, useCase: useCase}, nil
}

// ServeHTTP возвращает расшифрованное содержимое blob-секрета.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
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
		h.logger.Error().Err(err).Msg("get blob content failed")
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	defer output.Content.Close()

	w.Header().Set("Content-Type", output.Blob.ContentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+output.Blob.OriginalName+`"`)
	w.Header().Set("X-Checksum-SHA256", output.Blob.ChecksumSHA256)
	w.WriteHeader(http.StatusOK)
	if _, err = io.Copy(w, output.Content); err != nil {
		h.logger.Error().Err(err).Msg("write blob content failed")
	}
}
