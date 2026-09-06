package httputil

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/igor/gophkeeper/internal/modules/secrets/usecases"
)

func TestResponse(t *testing.T) {
	t.Run("Должен записать JSON response", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		WriteJSON(recorder, http.StatusCreated, map[string]string{"status": "ok"})

		require.Equal(t, http.StatusCreated, recorder.Code)
		assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
		assert.JSONEq(t, `{"status":"ok"}`, recorder.Body.String())
	})

	t.Run("Должен записать error response", func(t *testing.T) {
		recorder := httptest.NewRecorder()

		WriteError(recorder, http.StatusBadRequest, "bad_request", "bad request")

		require.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.JSONEq(t, `{"error":{"code":"bad_request","message":"bad request"}}`, recorder.Body.String())
	})
}

func TestWriteUseCaseError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
		wantResult bool
	}{
		{
			name:       "Должен записать 404 для отсутствующего secret",
			err:        usecases.ErrSecretNotFound,
			wantStatus: http.StatusNotFound,
			wantBody:   `{"error":{"code":"secret_not_found","message":"secret not found"}}`,
			wantResult: true,
		},
		{
			name:       "Должен записать 400 для ошибки типа secret",
			err:        usecases.ErrInvalidSecretType,
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":{"code":"validation_error","message":"request validation failed"}}`,
			wantResult: true,
		},
		{
			name:       "Должен записать 400 для ошибки JSON",
			err:        usecases.ErrInvalidJSON,
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":{"code":"validation_error","message":"request validation failed"}}`,
			wantResult: true,
		},
		{
			name:       "Должен вернуть false для неизвестной ошибки",
			err:        errors.New("unknown"),
			wantStatus: http.StatusOK,
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()

			got := WriteUseCaseError(recorder, tt.err)

			require.Equal(t, tt.wantResult, got)
			assert.Equal(t, tt.wantStatus, recorder.Code)
			if tt.wantBody != "" {
				assert.JSONEq(t, tt.wantBody, recorder.Body.String())
			}
		})
	}
}
