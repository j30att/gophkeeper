// Package httputil содержит общие helpers HTTP-ответов для secrets handlers.
package httputil

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/igor/gophkeeper/internal/modules/secrets/usecases"
)

// ErrorResponse описывает общий API-ответ с ошибкой.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// APIError описывает ошибку API.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON записывает response в формате JSON.
func WriteJSON(w http.ResponseWriter, status int, response any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

// WriteError записывает общий API-ответ с ошибкой.
func WriteError(w http.ResponseWriter, status int, code string, message string) {
	WriteJSON(w, status, ErrorResponse{Error: APIError{Code: code, Message: message}})
}

// WriteUseCaseError записывает HTTP-ошибку по ошибке usecase.
func WriteUseCaseError(w http.ResponseWriter, err error) bool {
	if errors.Is(err, usecases.ErrSecretNotFound) {
		WriteError(w, http.StatusNotFound, "secret_not_found", "secret not found")
		return true
	}
	if errors.Is(err, usecases.ErrBlobNotFound) {
		WriteError(w, http.StatusNotFound, "blob_not_found", "blob not found")
		return true
	}
	if errors.Is(err, usecases.ErrInvalidSecretType) || errors.Is(err, usecases.ErrInvalidJSON) {
		WriteError(w, http.StatusBadRequest, "validation_error", "request validation failed")
		return true
	}
	if errors.Is(err, usecases.ErrEmptyContent) {
		WriteError(w, http.StatusBadRequest, "validation_error", "content is required")
		return true
	}
	return false
}
