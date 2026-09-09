package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponse(t *testing.T) {
	t.Run(
		"Должен записать JSON response", func(t *testing.T) {
			recorder := httptest.NewRecorder()

			WriteJSON(recorder, http.StatusCreated, map[string]string{"status": "ok"})

			require.Equal(t, http.StatusCreated, recorder.Code)
			assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
			assert.JSONEq(t, `{"status":"ok"}`, recorder.Body.String())
		},
	)

	t.Run(
		"Должен записать error response", func(t *testing.T) {
			recorder := httptest.NewRecorder()

			WriteError(recorder, http.StatusBadRequest, "bad_request", "invalid request body")

			require.Equal(t, http.StatusBadRequest, recorder.Code)
			var response ErrorResponse
			require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
			assert.Equal(t, "bad_request", response.Error.Code)
			assert.Equal(t, "invalid request body", response.Error.Message)
		},
	)
}
