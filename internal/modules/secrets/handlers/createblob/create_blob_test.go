package createblob

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/igor/gophkeeper/internal/middleware"
	"github.com/igor/gophkeeper/internal/modules/auth/token"
	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
	"github.com/igor/gophkeeper/internal/modules/secrets/usecases"
)

type useCaseMock struct {
	input  usecases.BlobSecretInput
	output usecases.SecretOutput
	err    error
}

func (m *useCaseMock) Execute(_ context.Context, input usecases.BlobSecretInput) (usecases.SecretOutput, error) {
	m.input = input
	return m.output, m.err
}

type tokenValidatorMock struct {
	claims token.Claims
}

func (m *tokenValidatorMock) Validate(_ string) (token.Claims, error) {
	return m.claims, nil
}

func TestHandler(t *testing.T) {
	t.Run("Должен создать blob secret", func(t *testing.T) {
		userID := uuid.New()
		secretID := uuid.New()
		useCase := &useCaseMock{
			output: usecases.SecretOutput{
				ID:     secretID,
				UserID: userID,
				Type:   domain.SecretTypeText,
				Name:   "book",
				Blob:   &usecases.BlobOutput{OriginalName: "book.txt", Size: 7},
			},
		}
		handler := newTestHandler(t, useCase, userID)

		request := newMultipartRequest(t)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, request)

		require.Equal(t, http.StatusCreated, recorder.Code)
		assert.Equal(t, domain.SecretTypeText, useCase.input.Type)
		assert.Equal(t, "book", useCase.input.Name)
		content, err := io.ReadAll(useCase.input.Content)
		require.NoError(t, err)
		assert.Equal(t, "content", string(content))
		assert.JSONEq(t, `{"secret":{"id":"`+secretID.String()+`","user_id":"`+userID.String()+`","type":"text","name":"book","metadata":null,"blob":{"id":"00000000-0000-0000-0000-000000000000","original_name":"book.txt","content_type":"","size":7,"checksum_sha256":""},"version":0,"created_at":"0001-01-01T00:00:00Z","updated_at":"0001-01-01T00:00:00Z"}}`, recorder.Body.String())
	})

	t.Run("Должен вернуть 400 без file", func(t *testing.T) {
		handler := newTestHandler(t, &useCaseMock{}, uuid.New())
		request := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/blob", bytes.NewReader(nil))
		request.Header.Set("Content-Type", "multipart/form-data")
		request.Header.Set("Authorization", "Bearer token")

		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)

		assert.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func newTestHandler(t *testing.T, useCase UseCase, userID uuid.UUID) http.Handler {
	t.Helper()
	handler, err := New(zerolog.Nop(), useCase, validator.New(validator.WithRequiredStructEnabled()))
	require.NoError(t, err)
	return middleware.Auth(&tokenValidatorMock{claims: token.Claims{UserID: userID, Login: "igor"}})(handler)
}

func newMultipartRequest(t *testing.T) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("type", string(domain.SecretTypeText)))
	require.NoError(t, writer.WriteField("name", "book"))
	require.NoError(t, writer.WriteField("metadata", `{"kind":"book"}`))
	file, err := writer.CreateFormFile("file", "book.txt")
	require.NoError(t, err)
	_, err = file.Write([]byte("content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, "/api/v1/secrets/blob", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer token")
	return request
}
