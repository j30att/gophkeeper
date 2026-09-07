package bootstrap

import (
	"encoding/json"
	"net/http"

	authapi "github.com/igor/gophkeeper/pkg/api/generated/auth"
	infraapi "github.com/igor/gophkeeper/pkg/api/generated/infra"
	secretsapi "github.com/igor/gophkeeper/pkg/api/generated/secrets"
)

type generatedErrorResponse struct {
	Error generatedError `json:"error"`
}

type generatedError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func authStrictOptions() authapi.StrictHTTPServerOptions {
	return authapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  writeGeneratedRequestError,
		ResponseErrorHandlerFunc: writeGeneratedResponseError,
	}
}

func infraStrictOptions() infraapi.StrictHTTPServerOptions {
	return infraapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  writeGeneratedRequestError,
		ResponseErrorHandlerFunc: writeGeneratedResponseError,
	}
}

func secretsStrictOptions() secretsapi.StrictHTTPServerOptions {
	return secretsapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  writeGeneratedRequestError,
		ResponseErrorHandlerFunc: writeGeneratedResponseError,
	}
}

func writeGeneratedRequestError(w http.ResponseWriter, _ *http.Request, _ error) {
	writeGeneratedError(w, http.StatusBadRequest, "bad_request", "invalid request")
}

func writeGeneratedResponseError(w http.ResponseWriter, _ *http.Request, _ error) {
	writeGeneratedError(w, http.StatusInternalServerError, "internal_error", "internal server error")
}

func writeGeneratedError(w http.ResponseWriter, statusCode int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(
		generatedErrorResponse{
			Error: generatedError{
				Code:    code,
				Message: message,
			},
		},
	)
}
