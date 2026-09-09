// Package api содержит HTTP-клиент для GophKeeper server API.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SecretType описывает тип секрета на клиентской стороне.
type SecretType string

const (
	// SecretTypeCredentials описывает пару login/password.
	SecretTypeCredentials SecretType = "credentials"
	// SecretTypeCard описывает банковскую карту.
	SecretTypeCard SecretType = "card"
	// SecretTypeText описывает большой текстовый blob.
	SecretTypeText SecretType = "text"
	// SecretTypeBinary описывает бинарный blob.
	SecretTypeBinary SecretType = "binary"
)

// Client вызывает HTTP API сервера.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Secret описывает секрет, полученный от сервера.
type Secret struct {
	ID        uuid.UUID              `json:"id"`
	UserID    uuid.UUID              `json:"user_id"`
	Type      SecretType             `json:"type"`
	Name      string                 `json:"name"`
	Metadata  map[string]interface{} `json:"metadata"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	Blob      *Blob                  `json:"blob,omitempty"`
	Version   int                    `json:"version"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// Blob описывает метаданные blob-секрета.
type Blob struct {
	ID             uuid.UUID `json:"id"`
	OriginalName   string    `json:"original_name"`
	ContentType    string    `json:"content_type"`
	Size           int64     `json:"size"`
	ChecksumSHA256 string    `json:"checksum_sha256"`
}

// DeletedSecret описывает tombstone удаленного секрета.
type DeletedSecret struct {
	ID        uuid.UUID `json:"id"`
	Version   int       `json:"version"`
	DeletedAt time.Time `json:"deleted_at"`
}

// SyncResponse содержит изменения секретов с сервера.
type SyncResponse struct {
	ServerTime time.Time       `json:"server_time"`
	Secrets    []Secret        `json:"secrets"`
	Deleted    []DeletedSecret `json:"deleted"`
}

// SecretInput описывает structured-секрет для создания или обновления.
type SecretInput struct {
	Type            SecretType             `json:"type"`
	Name            string                 `json:"name"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Payload         map[string]interface{} `json:"payload"`
	ExpectedVersion int                    `json:"expected_version,omitempty"`
}

// BlobSecretInput описывает файл для создания blob-секрета.
type BlobSecretInput struct {
	Type         SecretType
	Name         string
	Metadata     string
	OriginalName string
	ContentType  string
	Content      io.Reader
}

// BlobContentInput описывает файл для замены содержимого blob-секрета.
type BlobContentInput struct {
	ExpectedVersion int
	OriginalName    string
	ContentType     string
	Content         io.Reader
}

// NewClient создает HTTP API client.
func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("empty server URL")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpClient}, nil
}

// Register регистрирует пользователя.
func (c *Client) Register(ctx context.Context, login string, password string) (string, error) {
	var response struct {
		UserID string `json:"user_id"`
	}
	if err := c.postPublic(ctx, "/api/v1/auth/register", authRequest{Login: login, Password: password}, &response); err != nil {
		return "", err
	}
	return response.UserID, nil
}

// Login получает JWT token.
func (c *Client) Login(ctx context.Context, login string, password string) (string, error) {
	var response struct {
		Token string `json:"token"`
	}
	if err := c.postPublic(ctx, "/api/v1/auth/login", authRequest{Login: login, Password: password}, &response); err != nil {
		return "", err
	}
	return response.Token, nil
}

// Sync получает полный snapshot или изменения после since.
func (c *Client) Sync(ctx context.Context, token string, since *time.Time) (SyncResponse, error) {
	path := "/api/v1/sync"
	if since != nil {
		path += "?since=" + since.UTC().Format(time.RFC3339)
	}
	var response SyncResponse
	if err := c.doJSON(ctx, http.MethodGet, path, token, nil, &response); err != nil {
		return SyncResponse{}, err
	}
	return response, nil
}

// GetSecret получает секрет по id.
func (c *Client) GetSecret(ctx context.Context, token string, id uuid.UUID) (Secret, error) {
	var response struct {
		Secret Secret `json:"secret"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/secrets/"+id.String(), token, nil, &response); err != nil {
		return Secret{}, err
	}
	return response.Secret, nil
}

// CreateSecret создает structured-секрет.
func (c *Client) CreateSecret(ctx context.Context, token string, input SecretInput) (Secret, error) {
	var response struct {
		Secret Secret `json:"secret"`
	}
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/secrets", token, input, &response); err != nil {
		return Secret{}, err
	}
	return response.Secret, nil
}

// UpdateSecret обновляет structured-секрет.
func (c *Client) UpdateSecret(ctx context.Context, token string, id uuid.UUID, input SecretInput) (Secret, error) {
	var response struct {
		Secret Secret `json:"secret"`
	}
	if err := c.doJSON(ctx, http.MethodPut, "/api/v1/secrets/"+id.String(), token, input, &response); err != nil {
		return Secret{}, err
	}
	return response.Secret, nil
}

// CreateBlobSecret создает blob-секрет через multipart upload.
func (c *Client) CreateBlobSecret(ctx context.Context, token string, input BlobSecretInput) (Secret, error) {
	var response struct {
		Secret Secret `json:"secret"`
	}
	if err := c.doMultipart(ctx, http.MethodPost, "/api/v1/secrets/blob", token, map[string]string{
		"type":     string(input.Type),
		"name":     input.Name,
		"metadata": input.Metadata,
	}, "file", input.OriginalName, input.ContentType, input.Content, &response); err != nil {
		return Secret{}, err
	}
	return response.Secret, nil
}

// DownloadBlobContent скачивает содержимое blob-секрета в writer.
func (c *Client) DownloadBlobContent(ctx context.Context, token string, id uuid.UUID, writer io.Writer) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/secrets/"+id.String()+"/content", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return decodeAPIError(response)
	}
	if _, err = io.Copy(writer, response.Body); err != nil {
		return fmt.Errorf("write blob content: %w", err)
	}
	return nil
}

// UpdateBlobContent заменяет содержимое blob-секрета через multipart upload.
func (c *Client) UpdateBlobContent(ctx context.Context, token string, id uuid.UUID, input BlobContentInput) (Secret, error) {
	var response struct {
		Secret Secret `json:"secret"`
	}
	if err := c.doMultipart(ctx, http.MethodPut, "/api/v1/secrets/"+id.String()+"/content", token, map[string]string{
		"expected_version": fmt.Sprintf("%d", input.ExpectedVersion),
	}, "file", input.OriginalName, input.ContentType, input.Content, &response); err != nil {
		return Secret{}, err
	}
	return response.Secret, nil
}

// DeleteSecret удаляет секрет.
func (c *Client) DeleteSecret(ctx context.Context, token string, id uuid.UUID) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/secrets/"+id.String(), token, nil, nil)
}

type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (c *Client) postPublic(ctx context.Context, path string, body any, output any) error {
	return c.doJSON(ctx, http.MethodPost, path, "", body, output)
}

func (c *Client) doJSON(ctx context.Context, method string, path string, token string, body any, output any) error {
	var requestBody io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		requestBody = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, requestBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return decodeAPIError(response)
	}
	if output == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}
	if err = json.NewDecoder(response.Body).Decode(output); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) doMultipart(
	ctx context.Context,
	method string,
	path string,
	token string,
	fields map[string]string,
	fileField string,
	fileName string,
	contentType string,
	content io.Reader,
	output any,
) error {
	if content == nil {
		return fmt.Errorf("empty file content")
	}
	reader, writer := io.Pipe()
	multipartWriter := multipart.NewWriter(writer)
	writeErr := make(chan error, 1)
	go func() {
		defer close(writeErr)
		for name, value := range fields {
			if err := multipartWriter.WriteField(name, value); err != nil {
				_ = writer.CloseWithError(err)
				writeErr <- err
				return
			}
		}
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fileField, fileName))
		if strings.TrimSpace(contentType) == "" {
			contentType = "application/octet-stream"
		}
		header.Set("Content-Type", contentType)
		part, err := multipartWriter.CreatePart(header)
		if err != nil {
			_ = writer.CloseWithError(err)
			writeErr <- err
			return
		}
		if _, err = io.Copy(part, content); err != nil {
			_ = writer.CloseWithError(err)
			writeErr <- err
			return
		}
		if err = multipartWriter.Close(); err != nil {
			_ = writer.CloseWithError(err)
			writeErr <- err
			return
		}
		writeErr <- writer.Close()
	}()

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		_ = reader.Close()
		return fmt.Errorf("create request: %w", err)
	}
	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		_ = reader.Close()
		return fmt.Errorf("send request: %w", err)
	}
	defer response.Body.Close()
	if err = <-writeErr; err != nil {
		return fmt.Errorf("write multipart request: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return decodeAPIError(response)
	}
	if output == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}
	if err = json.NewDecoder(response.Body).Decode(output); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func decodeAPIError(response *http.Response) error {
	var apiError struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&apiError); err != nil {
		return fmt.Errorf("api error: status %d", response.StatusCode)
	}
	if apiError.Error.Message == "" {
		return fmt.Errorf("api error: status %d", response.StatusCode)
	}
	return fmt.Errorf("api error: %s: %s", apiError.Error.Code, apiError.Error.Message)
}
