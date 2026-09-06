// Package createblob содержит HTTP-handler создания blob-секрета.
package createblob

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"github.com/igor/gophkeeper/internal/middleware"
	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
	"github.com/igor/gophkeeper/internal/modules/secrets/handlers/httputil"
	"github.com/igor/gophkeeper/internal/modules/secrets/usecases"
)

const maxMultipartMemory = 32 << 20

// UseCase создает blob-секрет.
type UseCase interface {
	Execute(ctx context.Context, input usecases.BlobSecretInput) (usecases.SecretOutput, error)
}

// Handler обрабатывает multipart-запрос создания blob-секрета.
type Handler struct {
	logger   zerolog.Logger
	useCase  UseCase
	validate *validator.Validate
}

type request struct {
	Type     domain.SecretType `validate:"required"`
	Name     string            `validate:"required"`
	Metadata json.RawMessage
}

type response struct {
	Secret usecases.SecretOutput `json:"secret"`
}

// New создает Handler.
func New(logger zerolog.Logger, useCase UseCase, validate *validator.Validate) (*Handler, error) {
	if useCase == nil {
		return nil, usecases.ErrEmptyDependency
	}
	if validate == nil {
		return nil, usecases.ErrEmptyDependency
	}
	return &Handler{logger: logger, useCase: useCase, validate: validate}, nil
}

// ServeHTTP создает blob-секрет из multipart/form-data.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}
	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "invalid multipart body")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "file is required")
		return
	}
	defer file.Close()

	inputRequest := request{
		Type:     domain.SecretType(r.FormValue("type")),
		Name:     r.FormValue("name"),
		Metadata: json.RawMessage(r.FormValue("metadata")),
	}
	if err = h.validate.Struct(inputRequest); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "validation_error", "request validation failed")
		return
	}

	contentType := header.Header.Get("Content-Type")
	output, err := h.useCase.Execute(
		r.Context(),
		usecases.BlobSecretInput{
			UserID:       userID,
			Type:         inputRequest.Type,
			Name:         inputRequest.Name,
			Metadata:     inputRequest.Metadata,
			OriginalName: header.Filename,
			ContentType:  contentType,
			Content:      file,
		},
	)
	if err != nil {
		if httputil.WriteUseCaseError(w, err) {
			return
		}
		h.logger.Error().Err(err).Msg("create blob secret failed")
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, response{Secret: output})
}
