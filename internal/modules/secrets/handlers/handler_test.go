package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/igor/gophkeeper/internal/middleware"
	authtoken "github.com/igor/gophkeeper/internal/modules/auth/token"
	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
	"github.com/igor/gophkeeper/internal/modules/secrets/usecases"
	"github.com/igor/gophkeeper/pkg/api/generated/secrets"
	secretsapi "github.com/igor/gophkeeper/pkg/api/generated/secrets"
)

type createUseCaseMock struct {
	output usecases.SecretOutput
	err    error
	input  usecases.SecretInput
}

func (m *createUseCaseMock) Execute(_ context.Context, input usecases.SecretInput) (usecases.SecretOutput, error) {
	m.input = input
	return m.output, m.err
}

type listUseCaseMock struct {
	output []usecases.SecretListItem
	err    error
	userID uuid.UUID
}

func (m *listUseCaseMock) Execute(_ context.Context, userID uuid.UUID) ([]usecases.SecretListItem, error) {
	m.userID = userID
	return m.output, m.err
}

type getUseCaseMock struct {
	output   usecases.SecretOutput
	err      error
	userID   uuid.UUID
	secretID uuid.UUID
}

func (m *getUseCaseMock) Execute(_ context.Context, userID uuid.UUID, secretID uuid.UUID) (usecases.SecretOutput, error) {
	m.userID = userID
	m.secretID = secretID
	return m.output, m.err
}

type updateUseCaseMock struct {
	output usecases.SecretOutput
	err    error
	input  usecases.UpdateSecretInput
}

func (m *updateUseCaseMock) Execute(_ context.Context, input usecases.UpdateSecretInput) (usecases.SecretOutput, error) {
	m.input = input
	return m.output, m.err
}

type deleteUseCaseMock struct {
	err      error
	userID   uuid.UUID
	secretID uuid.UUID
}

func (m *deleteUseCaseMock) Execute(_ context.Context, userID uuid.UUID, secretID uuid.UUID) error {
	m.userID = userID
	m.secretID = secretID
	return m.err
}

type createBlobUseCaseMock struct {
	output  usecases.SecretOutput
	err     error
	input   usecases.BlobSecretInput
	content string
}

func (m *createBlobUseCaseMock) Execute(_ context.Context, input usecases.BlobSecretInput) (usecases.SecretOutput, error) {
	m.input = input
	if input.Content != nil {
		content, _ := io.ReadAll(input.Content)
		m.content = string(content)
	}
	return m.output, m.err
}

type getBlobContentUseCaseMock struct {
	output   usecases.BlobContentOutput
	err      error
	userID   uuid.UUID
	secretID uuid.UUID
}

func (m *getBlobContentUseCaseMock) Execute(
	_ context.Context,
	userID uuid.UUID,
	secretID uuid.UUID,
) (usecases.BlobContentOutput, error) {
	m.userID = userID
	m.secretID = secretID
	return m.output, m.err
}

type updateBlobContentUseCaseMock struct {
	output  usecases.SecretOutput
	err     error
	input   usecases.BlobContentInput
	content string
}

func (m *updateBlobContentUseCaseMock) Execute(_ context.Context, input usecases.BlobContentInput) (usecases.SecretOutput, error) {
	m.input = input
	if input.Content != nil {
		content, _ := io.ReadAll(input.Content)
		m.content = string(content)
	}
	return m.output, m.err
}

type tokenValidatorMock struct {
	claims authtoken.Claims
	err    error
}

func (m *tokenValidatorMock) Validate(_ string) (authtoken.Claims, error) {
	return m.claims, m.err
}

func TestHandlerNew(t *testing.T) {
	t.Run(
		"Должен создать handler", func(t *testing.T) {
			handler, err := New(
				zerolog.Nop(),
				&createUseCaseMock{},
				&listUseCaseMock{},
				&getUseCaseMock{},
				&updateUseCaseMock{},
				&deleteUseCaseMock{},
				&createBlobUseCaseMock{},
				&getBlobContentUseCaseMock{},
				&updateBlobContentUseCaseMock{},
			)

			require.NoError(t, err)
			assert.NotNil(t, handler)
		},
	)

	t.Run(
		"Должен вернуть ошибку без зависимости", func(t *testing.T) {
			_, err := New(
				zerolog.Nop(),
				nil,
				&listUseCaseMock{},
				&getUseCaseMock{},
				&updateUseCaseMock{},
				&deleteUseCaseMock{},
				&createBlobUseCaseMock{},
				&getBlobContentUseCaseMock{},
				&updateBlobContentUseCaseMock{},
			)

			require.Error(t, err)
			assert.ErrorIs(t, err, usecases.ErrEmptyDependency)
		},
	)
}

func TestHandlerPostApiV1Secrets(t *testing.T) {
	t.Run(
		"Должен вернуть 201", func(t *testing.T) {
			userID := uuid.New()
			secretID := uuid.New()
			now := time.Now().UTC()
			createUseCase := &createUseCaseMock{
				output: usecases.SecretOutput{
					ID:        secretID,
					UserID:    userID,
					Type:      domain.SecretTypeCredentials,
					Name:      "github",
					Metadata:  json.RawMessage(`{"site":"github"}`),
					Payload:   json.RawMessage(`{"login":"igor"}`),
					Version:   1,
					CreatedAt: now,
					UpdatedAt: now,
				},
			}
			router := newTestRouter(t, userID, createUseCase, nil, nil, nil, nil, nil, nil, nil)
			request := newJSONRequest(t, http.MethodPost, "/api/v1/secrets", map[string]any{
				"type":     "credentials",
				"name":     "github",
				"metadata": map[string]any{"site": "github"},
				"payload":  map[string]any{"login": "igor"},
			})
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusCreated, recorder.Code)
			assert.Equal(t, userID, createUseCase.input.UserID)
			assert.Equal(t, domain.SecretTypeCredentials, createUseCase.input.Type)
			assert.JSONEq(t, `{"site":"github"}`, string(createUseCase.input.Metadata))
			assert.JSONEq(t, `{"login":"igor"}`, string(createUseCase.input.Payload))

			var response secrets.SecretResponse
			require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
			assert.Equal(t, secretID, uuid.UUID(response.Secret.Id))
		},
	)

	t.Run(
		"Должен вернуть 400 при ошибке валидации usecase", func(t *testing.T) {
			router := newTestRouter(
				t,
				uuid.New(),
				&createUseCaseMock{err: usecases.ErrInvalidSecretType},
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
			)
			request := newJSONRequest(t, http.MethodPost, "/api/v1/secrets", map[string]any{
				"type":    "credentials",
				"name":    "github",
				"payload": map[string]any{},
			})
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusBadRequest, recorder.Code)
		},
	)
}

func TestHandlerPostApiV1SecretsBlob(t *testing.T) {
	t.Run(
		"Должен вернуть 201", func(t *testing.T) {
			userID := uuid.New()
			now := time.Now().UTC()
			createBlobUseCase := &createBlobUseCaseMock{
				output: usecases.SecretOutput{
					ID:        uuid.New(),
					UserID:    userID,
					Type:      domain.SecretTypeText,
					Name:      "note",
					Metadata:  json.RawMessage(`{"kind":"book"}`),
					Payload:   json.RawMessage(`{}`),
					Version:   1,
					CreatedAt: now,
					UpdatedAt: now,
					Blob: &usecases.BlobOutput{
						ID:             uuid.New(),
						OriginalName:   "note.txt",
						ContentType:    "text/plain",
						Size:           12,
						ChecksumSHA256: "checksum",
					},
				},
			}
			router := newTestRouter(t, userID, nil, nil, nil, nil, nil, createBlobUseCase, nil, nil)
			request := newMultipartRequest(t, "/api/v1/secrets/blob", map[string]string{
				"type":     "text",
				"name":     "note",
				"metadata": `{"kind":"book"}`,
			}, "file", "note.txt", "text/plain", "hello world")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusCreated, recorder.Code)
			assert.Equal(t, userID, createBlobUseCase.input.UserID)
			assert.Equal(t, domain.SecretTypeText, createBlobUseCase.input.Type)
			assert.Equal(t, "note", createBlobUseCase.input.Name)
			assert.Equal(t, "note.txt", createBlobUseCase.input.OriginalName)
			assert.Equal(t, "text/plain", createBlobUseCase.input.ContentType)
			assert.Equal(t, "hello world", createBlobUseCase.content)
			assert.JSONEq(t, `{"kind":"book"}`, string(createBlobUseCase.input.Metadata))
		},
	)
}

func TestHandlerGetApiV1Secrets(t *testing.T) {
	t.Run(
		"Должен вернуть 200", func(t *testing.T) {
			userID := uuid.New()
			now := time.Now().UTC()
			listUseCase := &listUseCaseMock{
				output: []usecases.SecretListItem{
					{
						ID:        uuid.New(),
						Type:      domain.SecretTypeCredentials,
						Name:      "github",
						Metadata:  json.RawMessage(`{"site":"github"}`),
						Version:   1,
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			}
			router := newTestRouter(t, userID, nil, listUseCase, nil, nil, nil, nil, nil, nil)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/secrets", nil)
			request.Header.Set("Authorization", "Bearer token")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, userID, listUseCase.userID)
			var response secrets.SecretListResponse
			require.NoError(t, json.NewDecoder(recorder.Body).Decode(&response))
			require.Len(t, response.Secrets, 1)
			assert.Equal(t, "github", response.Secrets[0].Name)
		},
	)
}

func TestHandlerGetApiV1SecretsId(t *testing.T) {
	t.Run(
		"Должен вернуть 200", func(t *testing.T) {
			userID := uuid.New()
			secretID := uuid.New()
			now := time.Now().UTC()
			getUseCase := &getUseCaseMock{
				output: usecases.SecretOutput{
					ID:        secretID,
					UserID:    userID,
					Type:      domain.SecretTypeCredentials,
					Name:      "github",
					Metadata:  json.RawMessage(`{}`),
					Payload:   json.RawMessage(`{"login":"igor"}`),
					Version:   1,
					CreatedAt: now,
					UpdatedAt: now,
				},
			}
			router := newTestRouter(t, userID, nil, nil, getUseCase, nil, nil, nil, nil, nil)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/"+secretID.String(), nil)
			request.Header.Set("Authorization", "Bearer token")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, userID, getUseCase.userID)
			assert.Equal(t, secretID, getUseCase.secretID)
		},
	)

	t.Run(
		"Должен вернуть 404", func(t *testing.T) {
			router := newTestRouter(
				t,
				uuid.New(),
				nil,
				nil,
				&getUseCaseMock{err: usecases.ErrSecretNotFound},
				nil,
				nil,
				nil,
				nil,
				nil,
			)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/"+uuid.NewString(), nil)
			request.Header.Set("Authorization", "Bearer token")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusNotFound, recorder.Code)
		},
	)
}

func TestHandlerPutApiV1SecretsId(t *testing.T) {
	t.Run(
		"Должен вернуть 200", func(t *testing.T) {
			userID := uuid.New()
			secretID := uuid.New()
			now := time.Now().UTC()
			updateUseCase := &updateUseCaseMock{
				output: usecases.SecretOutput{
					ID:        secretID,
					UserID:    userID,
					Type:      domain.SecretTypeCard,
					Name:      "card",
					Metadata:  json.RawMessage(`{}`),
					Payload:   json.RawMessage(`{"number":"1234"}`),
					Version:   2,
					CreatedAt: now,
					UpdatedAt: now,
				},
			}
			router := newTestRouter(t, userID, nil, nil, nil, updateUseCase, nil, nil, nil, nil)
			request := newJSONRequest(t, http.MethodPut, "/api/v1/secrets/"+secretID.String(), map[string]any{
				"type":             "card",
				"name":             "card",
				"expected_version": 1,
				"payload":          map[string]any{"number": "1234"},
			})
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, userID, updateUseCase.input.UserID)
			assert.Equal(t, secretID, updateUseCase.input.ID)
			assert.Equal(t, domain.SecretTypeCard, updateUseCase.input.Type)
			assert.Equal(t, 1, updateUseCase.input.ExpectedVersion)
			assert.JSONEq(t, `{"number":"1234"}`, string(updateUseCase.input.Payload))
		},
	)
}

func TestHandlerDeleteApiV1SecretsId(t *testing.T) {
	t.Run(
		"Должен вернуть 204", func(t *testing.T) {
			userID := uuid.New()
			secretID := uuid.New()
			deleteUseCase := &deleteUseCaseMock{}
			router := newTestRouter(t, userID, nil, nil, nil, nil, deleteUseCase, nil, nil, nil)
			request := httptest.NewRequest(http.MethodDelete, "/api/v1/secrets/"+secretID.String(), nil)
			request.Header.Set("Authorization", "Bearer token")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusNoContent, recorder.Code)
			assert.Equal(t, userID, deleteUseCase.userID)
			assert.Equal(t, secretID, deleteUseCase.secretID)
		},
	)
}

func TestHandlerGetApiV1SecretsIdContent(t *testing.T) {
	t.Run(
		"Должен вернуть 200 и stream содержимого", func(t *testing.T) {
			userID := uuid.New()
			secretID := uuid.New()
			getBlobContentUseCase := &getBlobContentUseCaseMock{
				output: usecases.BlobContentOutput{
					Blob: usecases.BlobOutput{
						Size:           int64(len("content")),
						ChecksumSHA256: "checksum",
					},
					Content: io.NopCloser(bytes.NewBufferString("content")),
				},
			}
			router := newTestRouter(t, userID, nil, nil, nil, nil, nil, nil, getBlobContentUseCase, nil)
			request := httptest.NewRequest(http.MethodGet, "/api/v1/secrets/"+secretID.String()+"/content", nil)
			request.Header.Set("Authorization", "Bearer token")
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, "checksum", recorder.Header().Get("X-Checksum-SHA256"))
			assert.Equal(t, "content", recorder.Body.String())
			assert.Equal(t, userID, getBlobContentUseCase.userID)
			assert.Equal(t, secretID, getBlobContentUseCase.secretID)
		},
	)
}

func TestHandlerPutApiV1SecretsIdContent(t *testing.T) {
	t.Run(
		"Должен вернуть 200", func(t *testing.T) {
			userID := uuid.New()
			secretID := uuid.New()
			now := time.Now().UTC()
			updateBlobContentUseCase := &updateBlobContentUseCaseMock{
				output: usecases.SecretOutput{
					ID:        secretID,
					UserID:    userID,
					Type:      domain.SecretTypeText,
					Name:      "note",
					Metadata:  json.RawMessage(`{}`),
					Payload:   json.RawMessage(`{}`),
					Version:   2,
					CreatedAt: now,
					UpdatedAt: now,
					Blob: &usecases.BlobOutput{
						ID:             uuid.New(),
						OriginalName:   "note.txt",
						ContentType:    "text/plain",
						Size:           11,
						ChecksumSHA256: "checksum",
					},
				},
			}
			router := newTestRouter(t, userID, nil, nil, nil, nil, nil, nil, nil, updateBlobContentUseCase)
			request := newMultipartRequest(t, "/api/v1/secrets/"+secretID.String()+"/content", map[string]string{
				"expected_version": "1",
			}, "file", "note.txt", "text/plain", "new content")
			request.Method = http.MethodPut
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, userID, updateBlobContentUseCase.input.UserID)
			assert.Equal(t, secretID, updateBlobContentUseCase.input.ID)
			assert.Equal(t, 1, updateBlobContentUseCase.input.ExpectedVersion)
			assert.Equal(t, "note.txt", updateBlobContentUseCase.input.OriginalName)
			assert.Equal(t, "text/plain", updateBlobContentUseCase.input.ContentType)
			assert.Equal(t, "new content", updateBlobContentUseCase.content)
		},
	)

	t.Run(
		"Должен вернуть 409", func(t *testing.T) {
			router := newTestRouter(
				t,
				uuid.New(),
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				&updateBlobContentUseCaseMock{err: usecases.ErrSecretVersionConflict},
			)
			request := newMultipartRequest(t, "/api/v1/secrets/"+uuid.NewString()+"/content", map[string]string{
				"expected_version": "1",
			}, "file", "note.txt", "text/plain", "new content")
			request.Method = http.MethodPut
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusConflict, recorder.Code)
		},
	)
}

func TestHandlerUnauthorized(t *testing.T) {
	router := newTestRouter(t, uuid.New(), nil, nil, nil, nil, nil, nil, nil, nil)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/secrets", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func newTestRouter(
	t *testing.T,
	userID uuid.UUID,
	createUseCase CreateUseCase,
	listUseCase ListUseCase,
	getUseCase GetUseCase,
	updateUseCase UpdateUseCase,
	deleteUseCase DeleteUseCase,
	createBlobUseCase CreateBlobUseCase,
	getBlobContentUseCase GetBlobContentUseCase,
	updateBlobContentUseCase UpdateBlobContentUseCase,
) http.Handler {
	t.Helper()
	if createUseCase == nil {
		createUseCase = &createUseCaseMock{}
	}
	if listUseCase == nil {
		listUseCase = &listUseCaseMock{}
	}
	if getUseCase == nil {
		getUseCase = &getUseCaseMock{}
	}
	if updateUseCase == nil {
		updateUseCase = &updateUseCaseMock{}
	}
	if deleteUseCase == nil {
		deleteUseCase = &deleteUseCaseMock{}
	}
	if createBlobUseCase == nil {
		createBlobUseCase = &createBlobUseCaseMock{}
	}
	if getBlobContentUseCase == nil {
		getBlobContentUseCase = &getBlobContentUseCaseMock{}
	}
	if updateBlobContentUseCase == nil {
		updateBlobContentUseCase = &updateBlobContentUseCaseMock{}
	}
	handler, err := New(
		zerolog.Nop(),
		createUseCase,
		listUseCase,
		getUseCase,
		updateUseCase,
		deleteUseCase,
		createBlobUseCase,
		getBlobContentUseCase,
		updateBlobContentUseCase,
	)
	require.NoError(t, err)
	router := chi.NewRouter()
	router.Use(middleware.Auth(&tokenValidatorMock{claims: authtoken.Claims{UserID: userID, Login: "igor"}}))
	secretsapi.HandlerFromMux(secretsapi.NewStrictHandler(handler, nil), router)
	return router
}

func newJSONRequest(t *testing.T, method string, target string, body any) *http.Request {
	t.Helper()
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	request := httptest.NewRequest(method, target, bytes.NewReader(payload))
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("Content-Type", "application/json")
	return request
}

func newMultipartRequest(
	t *testing.T,
	target string,
	fields map[string]string,
	fileField string,
	fileName string,
	contentType string,
	content string,
) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range fields {
		require.NoError(t, writer.WriteField(name, value))
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="`+fileField+`"; filename="`+fileName+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	request := httptest.NewRequest(http.MethodPost, target, &body)
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}
