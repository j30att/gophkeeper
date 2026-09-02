// Package httputil содержит общие helpers HTTP-ответов для auth handlers.
package httputil

import (
	"encoding/json"
	"net/http"
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
