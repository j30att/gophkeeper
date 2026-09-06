package getcontent

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/igor/gophkeeper/internal/middleware"
	"github.com/igor/gophkeeper/internal/modules/auth/token"
	"github.com/igor/gophkeeper/internal/modules/secrets/usecases"
)

type useCaseMock struct {
	output usecases.BlobContentOutput
	err    error
}

func (m *useCaseMock) Execute(_ context.Context, _ uuid.UUID, _ uuid.UUID) (usecases.BlobContentOutput, error) {
	return m.output, m.err
}

type tokenValidatorMock struct {
	claims token.Claims
}

func (m *tokenValidatorMock) Validate(_ string) (token.Claims, error) {
	return m.claims, nil
}

func TestHandler(t *testing.T) {
	t.Run("Должен вернуть content", func(t *testing.T) {
		userID := uuid.New()
		handler := newTestHandler(
			t,
			&useCaseMock{
				output: usecases.BlobContentOutput{
					Blob: usecases.BlobOutput{
						OriginalName:   "book.txt",
						ContentType:    "text/plain",
						ChecksumSHA256: "checksum",
					},
					Content: io.NopCloser(strings.NewReader("content")),
				},
			},
			userID,
		)
		request := newRequest(uuid.New().String())
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusOK, recorder.Code)
		assert.Equal(t, "text/plain", recorder.Header().Get("Content-Type"))
		assert.Equal(t, "checksum", recorder.Header().Get("X-Checksum-SHA256"))
		assert.Equal(t, "content", recorder.Body.String())
	})

	t.Run("Должен вернуть 400 при невалидном id", func(t *testing.T) {
		handler := newTestHandler(t, &useCaseMock{}, uuid.New())
		request := newRequest("bad")
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func newTestHandler(t *testing.T, useCase UseCase, userID uuid.UUID) http.Handler {
	t.Helper()
	handler, err := New(zerolog.Nop(), useCase)
	require.NoError(t, err)
	return middleware.Auth(&tokenValidatorMock{claims: token.Claims{UserID: userID, Login: "igor"}})(handler)
}

func newRequest(secretID string) *http.Request {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", secretID)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/"+secretID+"/content", nil)
	request.Header.Set("Authorization", "Bearer token")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	return request
}
