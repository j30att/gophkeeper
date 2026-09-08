// Package handlers содержит generated HTTP-handlers secrets API.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"strconv"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/igor/gophkeeper/internal/middleware"
	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
	"github.com/igor/gophkeeper/internal/modules/secrets/usecases"
	"github.com/igor/gophkeeper/pkg/api/generated/secrets"
)

// CreateUseCase создает structured-секрет.
type CreateUseCase interface {
	Execute(ctx context.Context, input usecases.SecretInput) (usecases.SecretOutput, error)
}

// ListUseCase возвращает список секретов.
type ListUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID) ([]usecases.SecretListItem, error)
}

// GetUseCase возвращает секрет по id.
type GetUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (usecases.SecretOutput, error)
}

// UpdateUseCase обновляет structured-секрет.
type UpdateUseCase interface {
	Execute(ctx context.Context, input usecases.UpdateSecretInput) (usecases.SecretOutput, error)
}

// DeleteUseCase удаляет секрет.
type DeleteUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) error
}

// CreateBlobUseCase создает blob-секрет.
type CreateBlobUseCase interface {
	Execute(ctx context.Context, input usecases.BlobSecretInput) (usecases.SecretOutput, error)
}

// GetBlobContentUseCase возвращает содержимое blob-секрета.
type GetBlobContentUseCase interface {
	Execute(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (usecases.BlobContentOutput, error)
}

// UpdateBlobContentUseCase заменяет содержимое blob-секрета.
type UpdateBlobContentUseCase interface {
	Execute(ctx context.Context, input usecases.BlobContentInput) (usecases.SecretOutput, error)
}

// SyncUseCase возвращает изменения секретов.
type SyncUseCase interface {
	Execute(ctx context.Context, input usecases.SyncInput) (usecases.SyncOutput, error)
}

// Handler реализует generated secrets.StrictServerInterface.
type Handler struct {
	logger            zerolog.Logger
	create            CreateUseCase
	list              ListUseCase
	get               GetUseCase
	update            UpdateUseCase
	delete            DeleteUseCase
	createBlob        CreateBlobUseCase
	getBlobStream     GetBlobContentUseCase
	updateBlobContent UpdateBlobContentUseCase
	sync              SyncUseCase
}

// New создает Handler.
func New(
	logger zerolog.Logger,
	create CreateUseCase,
	list ListUseCase,
	get GetUseCase,
	update UpdateUseCase,
	delete DeleteUseCase,
	createBlob CreateBlobUseCase,
	getBlobStream GetBlobContentUseCase,
	updateBlobContent UpdateBlobContentUseCase,
	sync SyncUseCase,
) (*Handler, error) {
	if create == nil {
		return nil, fmt.Errorf("%w: create", usecases.ErrEmptyDependency)
	}
	if list == nil {
		return nil, fmt.Errorf("%w: list", usecases.ErrEmptyDependency)
	}
	if get == nil {
		return nil, fmt.Errorf("%w: get", usecases.ErrEmptyDependency)
	}
	if update == nil {
		return nil, fmt.Errorf("%w: update", usecases.ErrEmptyDependency)
	}
	if delete == nil {
		return nil, fmt.Errorf("%w: delete", usecases.ErrEmptyDependency)
	}
	if createBlob == nil {
		return nil, fmt.Errorf("%w: createBlob", usecases.ErrEmptyDependency)
	}
	if getBlobStream == nil {
		return nil, fmt.Errorf("%w: getBlobStream", usecases.ErrEmptyDependency)
	}
	if updateBlobContent == nil {
		return nil, fmt.Errorf("%w: updateBlobContent", usecases.ErrEmptyDependency)
	}
	if sync == nil {
		return nil, fmt.Errorf("%w: sync", usecases.ErrEmptyDependency)
	}
	return &Handler{
		logger:            logger,
		create:            create,
		list:              list,
		get:               get,
		update:            update,
		delete:            delete,
		createBlob:        createBlob,
		getBlobStream:     getBlobStream,
		updateBlobContent: updateBlobContent,
		sync:              sync,
	}, nil
}

// GetApiV1Secrets возвращает список секретов пользователя.
func (h *Handler) GetApiV1Secrets(
	ctx context.Context,
	_ secrets.GetApiV1SecretsRequestObject,
) (secrets.GetApiV1SecretsResponseObject, error) {
	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return secrets.GetApiV1Secrets401JSONResponse(errorResponse("unauthorized", "authorization required")), nil
	}
	output, err := h.list.Execute(ctx, userID)
	if err != nil {
		h.logger.Error().Err(err).Msg("list secrets failed")
		return secrets.GetApiV1Secrets500JSONResponse(errorResponse("internal_error", "internal server error")), nil
	}
	return secrets.GetApiV1Secrets200JSONResponse{Secrets: toSecretList(output)}, nil
}

// PostApiV1Secrets создает structured-секрет.
func (h *Handler) PostApiV1Secrets(
	ctx context.Context,
	request secrets.PostApiV1SecretsRequestObject,
) (secrets.PostApiV1SecretsResponseObject, error) {
	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return secrets.PostApiV1Secrets401JSONResponse(errorResponse("unauthorized", "authorization required")), nil
	}
	if request.Body == nil {
		return secrets.PostApiV1Secrets400JSONResponse(errorResponse("bad_request", "invalid request body")), nil
	}
	metadata, err := mapToRawMessage(request.Body.Metadata)
	if err != nil {
		return secrets.PostApiV1Secrets400JSONResponse(errorResponse("bad_request", "invalid metadata")), nil
	}
	payload, err := mapToRawMessage(&request.Body.Payload)
	if err != nil {
		return secrets.PostApiV1Secrets400JSONResponse(errorResponse("bad_request", "invalid payload")), nil
	}

	output, err := h.create.Execute(
		ctx,
		usecases.SecretInput{
			UserID:   userID,
			Type:     domain.SecretType(request.Body.Type),
			Name:     request.Body.Name,
			Metadata: metadata,
			Payload:  payload,
		},
	)
	if err != nil {
		return h.createErrorResponse(err)
	}
	return secrets.PostApiV1Secrets201JSONResponse{Secret: toSecret(output)}, nil
}

// PostApiV1SecretsBlob создает blob-секрет.
func (h *Handler) PostApiV1SecretsBlob(
	ctx context.Context,
	request secrets.PostApiV1SecretsBlobRequestObject,
) (secrets.PostApiV1SecretsBlobResponseObject, error) {
	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return secrets.PostApiV1SecretsBlob401JSONResponse(errorResponse("unauthorized", "authorization required")), nil
	}
	input, err := readBlobMultipart(userID, request.Body)
	if err != nil {
		return secrets.PostApiV1SecretsBlob400JSONResponse(errorResponse("bad_request", err.Error())), nil
	}

	output, err := h.createBlob.Execute(ctx, input)
	if err != nil {
		return h.createBlobErrorResponse(err)
	}
	return secrets.PostApiV1SecretsBlob201JSONResponse{Secret: toSecret(output)}, nil
}

// DeleteApiV1SecretsId удаляет секрет.
func (h *Handler) DeleteApiV1SecretsId(
	ctx context.Context,
	request secrets.DeleteApiV1SecretsIdRequestObject,
) (secrets.DeleteApiV1SecretsIdResponseObject, error) {
	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return secrets.DeleteApiV1SecretsId401JSONResponse(errorResponse("unauthorized", "authorization required")), nil
	}
	if err := h.delete.Execute(ctx, userID, request.Id); err != nil {
		if errors.Is(err, usecases.ErrSecretNotFound) {
			return secrets.DeleteApiV1SecretsId404JSONResponse(errorResponse("secret_not_found", "secret not found")), nil
		}
		h.logger.Error().Err(err).Msg("delete secret failed")
		return nil, err
	}
	return secrets.DeleteApiV1SecretsId204Response{}, nil
}

// GetApiV1SecretsId возвращает секрет по id.
func (h *Handler) GetApiV1SecretsId(
	ctx context.Context,
	request secrets.GetApiV1SecretsIdRequestObject,
) (secrets.GetApiV1SecretsIdResponseObject, error) {
	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return secrets.GetApiV1SecretsId401JSONResponse(errorResponse("unauthorized", "authorization required")), nil
	}
	output, err := h.get.Execute(ctx, userID, request.Id)
	if err != nil {
		if errors.Is(err, usecases.ErrSecretNotFound) || errors.Is(err, usecases.ErrBlobNotFound) {
			return secrets.GetApiV1SecretsId404JSONResponse(errorResponse("secret_not_found", "secret not found")), nil
		}
		h.logger.Error().Err(err).Msg("get secret failed")
		return nil, err
	}
	return secrets.GetApiV1SecretsId200JSONResponse{Secret: toSecret(output)}, nil
}

// PutApiV1SecretsId обновляет structured-секрет.
func (h *Handler) PutApiV1SecretsId(
	ctx context.Context,
	request secrets.PutApiV1SecretsIdRequestObject,
) (secrets.PutApiV1SecretsIdResponseObject, error) {
	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return secrets.PutApiV1SecretsId401JSONResponse(errorResponse("unauthorized", "authorization required")), nil
	}
	if request.Body == nil {
		return secrets.PutApiV1SecretsId400JSONResponse(errorResponse("bad_request", "invalid request body")), nil
	}
	if request.Body.ExpectedVersion < 1 {
		return secrets.PutApiV1SecretsId400JSONResponse(errorResponse("bad_request", "expected_version is required")), nil
	}
	metadata, err := mapToRawMessage(request.Body.Metadata)
	if err != nil {
		return secrets.PutApiV1SecretsId400JSONResponse(errorResponse("bad_request", "invalid metadata")), nil
	}
	payload, err := mapToRawMessage(&request.Body.Payload)
	if err != nil {
		return secrets.PutApiV1SecretsId400JSONResponse(errorResponse("bad_request", "invalid payload")), nil
	}

	output, err := h.update.Execute(
		ctx,
		usecases.UpdateSecretInput{
			UserID:          userID,
			ID:              request.Id,
			Type:            domain.SecretType(request.Body.Type),
			Name:            request.Body.Name,
			Metadata:        metadata,
			Payload:         payload,
			ExpectedVersion: request.Body.ExpectedVersion,
		},
	)
	if err != nil {
		return h.updateErrorResponse(err)
	}
	return secrets.PutApiV1SecretsId200JSONResponse{Secret: toSecret(output)}, nil
}

// GetApiV1SecretsIdContent возвращает содержимое blob-секрета.
func (h *Handler) GetApiV1SecretsIdContent(
	ctx context.Context,
	request secrets.GetApiV1SecretsIdContentRequestObject,
) (secrets.GetApiV1SecretsIdContentResponseObject, error) {
	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return secrets.GetApiV1SecretsIdContent401JSONResponse(errorResponse("unauthorized", "authorization required")), nil
	}
	output, err := h.getBlobStream.Execute(ctx, userID, request.Id)
	if err != nil {
		if errors.Is(err, usecases.ErrBlobNotFound) || errors.Is(err, usecases.ErrSecretNotFound) {
			return secrets.GetApiV1SecretsIdContent404JSONResponse(errorResponse("blob_not_found", "blob not found")), nil
		}
		h.logger.Error().Err(err).Msg("get blob content failed")
		return nil, err
	}
	return secrets.GetApiV1SecretsIdContent200ApplicationoctetStreamResponse{
		Body:          output.Content,
		ContentLength: output.Blob.Size,
		Headers: secrets.GetApiV1SecretsIdContent200ResponseHeaders{
			ContentDisposition: mime.FormatMediaType(
				"attachment",
				map[string]string{"filename": output.Blob.OriginalName},
			),
			XChecksumSHA256: output.Blob.ChecksumSHA256,
		},
	}, nil
}

// PutApiV1SecretsIdContent заменяет содержимое blob-секрета.
func (h *Handler) PutApiV1SecretsIdContent(
	ctx context.Context,
	request secrets.PutApiV1SecretsIdContentRequestObject,
) (secrets.PutApiV1SecretsIdContentResponseObject, error) {
	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return secrets.PutApiV1SecretsIdContent401JSONResponse(errorResponse("unauthorized", "authorization required")), nil
	}
	input, err := readBlobContentMultipart(userID, request.Id, request.Body)
	if err != nil {
		return secrets.PutApiV1SecretsIdContent400JSONResponse(errorResponse("bad_request", err.Error())), nil
	}
	output, err := h.updateBlobContent.Execute(ctx, input)
	if err != nil {
		return h.updateBlobContentErrorResponse(err)
	}
	return secrets.PutApiV1SecretsIdContent200JSONResponse{Secret: toSecret(output)}, nil
}

// GetApiV1Sync возвращает изменения секретов пользователя для синхронизации.
func (h *Handler) GetApiV1Sync(
	ctx context.Context,
	request secrets.GetApiV1SyncRequestObject,
) (secrets.GetApiV1SyncResponseObject, error) {
	userID, ok := middleware.UserIDFromContext(ctx)
	if !ok {
		return secrets.GetApiV1Sync401JSONResponse(errorResponse("unauthorized", "authorization required")), nil
	}
	output, err := h.sync.Execute(ctx, usecases.SyncInput{UserID: userID, Since: request.Params.Since})
	if err != nil {
		h.logger.Error().Err(err).Msg("sync secrets failed")
		return secrets.GetApiV1Sync500JSONResponse(errorResponse("internal_error", "internal server error")), nil
	}
	return secrets.GetApiV1Sync200JSONResponse{
		ServerTime: output.ServerTime,
		Secrets:    toSyncSecrets(output.Secrets),
		Deleted:    toDeletedSecrets(output.Deleted),
	}, nil
}

func (h *Handler) createErrorResponse(err error) (secrets.PostApiV1SecretsResponseObject, error) {
	if errors.Is(err, usecases.ErrInvalidSecretType) || errors.Is(err, usecases.ErrInvalidJSON) {
		return secrets.PostApiV1Secrets400JSONResponse(errorResponse("validation_error", "request validation failed")), nil
	}
	h.logger.Error().Err(err).Msg("create secret failed")
	return secrets.PostApiV1Secrets500JSONResponse(errorResponse("internal_error", "internal server error")), nil
}

func (h *Handler) createBlobErrorResponse(err error) (secrets.PostApiV1SecretsBlobResponseObject, error) {
	if errors.Is(err, usecases.ErrInvalidSecretType) || errors.Is(err, usecases.ErrInvalidJSON) ||
		errors.Is(err, usecases.ErrEmptyContent) {
		return secrets.PostApiV1SecretsBlob400JSONResponse(errorResponse("validation_error", "request validation failed")), nil
	}
	h.logger.Error().Err(err).Msg("create blob secret failed")
	return secrets.PostApiV1SecretsBlob500JSONResponse(errorResponse("internal_error", "internal server error")), nil
}

func (h *Handler) updateErrorResponse(err error) (secrets.PutApiV1SecretsIdResponseObject, error) {
	if errors.Is(err, usecases.ErrSecretNotFound) {
		return secrets.PutApiV1SecretsId404JSONResponse(errorResponse("secret_not_found", "secret not found")), nil
	}
	if errors.Is(err, usecases.ErrSecretVersionConflict) {
		return secrets.PutApiV1SecretsId409JSONResponse(errorResponse("version_conflict", "secret version conflict")), nil
	}
	if errors.Is(err, usecases.ErrInvalidSecretType) || errors.Is(err, usecases.ErrInvalidJSON) {
		return secrets.PutApiV1SecretsId400JSONResponse(errorResponse("validation_error", "request validation failed")), nil
	}
	h.logger.Error().Err(err).Msg("update secret failed")
	return nil, err
}

func (h *Handler) updateBlobContentErrorResponse(err error) (secrets.PutApiV1SecretsIdContentResponseObject, error) {
	if errors.Is(err, usecases.ErrSecretNotFound) || errors.Is(err, usecases.ErrBlobNotFound) {
		return secrets.PutApiV1SecretsIdContent404JSONResponse(errorResponse("blob_not_found", "blob not found")), nil
	}
	if errors.Is(err, usecases.ErrSecretVersionConflict) {
		return secrets.PutApiV1SecretsIdContent409JSONResponse(errorResponse("version_conflict", "secret version conflict")), nil
	}
	if errors.Is(err, usecases.ErrInvalidSecretType) || errors.Is(err, usecases.ErrEmptyContent) {
		return secrets.PutApiV1SecretsIdContent400JSONResponse(errorResponse("validation_error", "request validation failed")), nil
	}
	h.logger.Error().Err(err).Msg("update blob content failed")
	return secrets.PutApiV1SecretsIdContent500JSONResponse(errorResponse("internal_error", "internal server error")), nil
}

func readBlobMultipart(userID uuid.UUID, reader *multipart.Reader) (usecases.BlobSecretInput, error) {
	if reader == nil {
		return usecases.BlobSecretInput{}, errors.New("invalid multipart body")
	}
	var input usecases.BlobSecretInput
	input.UserID = userID
	for {
		part, err := reader.NextPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return usecases.BlobSecretInput{}, fmt.Errorf("read multipart: %w", err)
		}
		switch part.FormName() {
		case "type":
			value, err := io.ReadAll(part)
			if err != nil {
				return usecases.BlobSecretInput{}, fmt.Errorf("read type: %w", err)
			}
			input.Type = domain.SecretType(string(value))
		case "name":
			value, err := io.ReadAll(part)
			if err != nil {
				return usecases.BlobSecretInput{}, fmt.Errorf("read name: %w", err)
			}
			input.Name = string(value)
		case "metadata":
			value, err := io.ReadAll(part)
			if err != nil {
				return usecases.BlobSecretInput{}, fmt.Errorf("read metadata: %w", err)
			}
			input.Metadata = value
		case "file":
			input.OriginalName = part.FileName()
			input.ContentType = part.Header.Get("Content-Type")
			input.Content = part
			return input, nil
		}
	}
	return usecases.BlobSecretInput{}, errors.New("file is required")
}

func readBlobContentMultipart(userID uuid.UUID, secretID uuid.UUID, reader *multipart.Reader) (usecases.BlobContentInput, error) {
	if reader == nil {
		return usecases.BlobContentInput{}, errors.New("invalid multipart body")
	}
	var input usecases.BlobContentInput
	input.UserID = userID
	input.ID = secretID
	for {
		part, err := reader.NextPart()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return usecases.BlobContentInput{}, fmt.Errorf("read multipart: %w", err)
		}
		switch part.FormName() {
		case "expected_version":
			value, err := io.ReadAll(part)
			if err != nil {
				return usecases.BlobContentInput{}, fmt.Errorf("read expected_version: %w", err)
			}
			expectedVersion, err := strconv.Atoi(string(value))
			if err != nil {
				return usecases.BlobContentInput{}, errors.New("expected_version must be integer")
			}
			input.ExpectedVersion = expectedVersion
		case "file":
			input.OriginalName = part.FileName()
			input.ContentType = part.Header.Get("Content-Type")
			input.Content = part
			if input.ExpectedVersion < 1 {
				return usecases.BlobContentInput{}, errors.New("expected_version is required")
			}
			return input, nil
		}
	}
	return usecases.BlobContentInput{}, errors.New("file is required")
}

func mapToRawMessage(value *map[string]interface{}) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func rawMessageToMap(value json.RawMessage) map[string]interface{} {
	if len(value) == 0 {
		return map[string]interface{}{}
	}
	var result map[string]interface{}
	if err := json.Unmarshal(value, &result); err != nil {
		return map[string]interface{}{}
	}
	return result
}

func toSecret(output usecases.SecretOutput) secrets.Secret {
	secret := secrets.Secret{
		Id:        output.ID,
		UserId:    output.UserID,
		Type:      secrets.SecretType(output.Type),
		Name:      output.Name,
		Metadata:  rawMessageToMap(output.Metadata),
		Version:   output.Version,
		CreatedAt: output.CreatedAt,
		UpdatedAt: output.UpdatedAt,
	}
	if len(output.Payload) > 0 {
		payload := rawMessageToMap(output.Payload)
		secret.Payload = &payload
	}
	if output.Blob != nil {
		secret.Blob = ptrBlob(toBlob(*output.Blob))
	}
	return secret
}

func toSecretList(items []usecases.SecretListItem) []secrets.SecretListItem {
	result := make([]secrets.SecretListItem, 0, len(items))
	for _, item := range items {
		listItem := secrets.SecretListItem{
			Id:        item.ID,
			Type:      secrets.SecretType(item.Type),
			Name:      item.Name,
			Metadata:  rawMessageToMap(item.Metadata),
			Version:   item.Version,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		}
		if item.Blob != nil {
			listItem.Blob = ptrBlob(toBlob(*item.Blob))
		}
		result = append(result, listItem)
	}
	return result
}

func toBlob(blob usecases.BlobOutput) secrets.Blob {
	return secrets.Blob{
		Id:             blob.ID,
		OriginalName:   blob.OriginalName,
		ContentType:    blob.ContentType,
		Size:           blob.Size,
		ChecksumSha256: blob.ChecksumSHA256,
	}
}

func ptrBlob(blob secrets.Blob) *secrets.Blob {
	return &blob
}

func toSyncSecrets(items []usecases.SecretOutput) []secrets.Secret {
	result := make([]secrets.Secret, 0, len(items))
	for _, item := range items {
		result = append(result, toSecret(item))
	}
	return result
}

func toDeletedSecrets(items []usecases.DeletedSecretOutput) []secrets.DeletedSecret {
	result := make([]secrets.DeletedSecret, 0, len(items))
	for _, item := range items {
		result = append(
			result,
			secrets.DeletedSecret{
				Id:        item.ID,
				Version:   item.Version,
				DeletedAt: item.DeletedAt,
			},
		)
	}
	return result
}

func errorResponse(code string, message string) secrets.ErrorResponse {
	return secrets.ErrorResponse{
		Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{Code: code, Message: message},
	}
}
