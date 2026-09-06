package usecases

import (
	"encoding/json"
	"fmt"

	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
)

func validateStructuredInput(secretType domain.SecretType, metadata json.RawMessage, payload json.RawMessage) error {
	if !domain.IsStructuredType(secretType) {
		return ErrInvalidSecretType
	}
	if err := validateJSON(metadata, []byte("{}")); err != nil {
		return fmt.Errorf("%w: metadata", err)
	}
	if err := validateJSON(payload, nil); err != nil {
		return fmt.Errorf("%w: payload", err)
	}
	return nil
}

func validateBlobInput(secretType domain.SecretType, metadata json.RawMessage) error {
	if !domain.IsBlobType(secretType) {
		return ErrInvalidSecretType
	}
	if err := validateJSON(metadata, []byte("{}")); err != nil {
		return fmt.Errorf("%w: metadata", err)
	}
	return nil
}

func normalizeJSON(value json.RawMessage, fallback []byte) json.RawMessage {
	if len(value) == 0 && fallback != nil {
		return json.RawMessage(fallback)
	}
	return value
}

func validateJSON(value json.RawMessage, fallback []byte) error {
	normalized := normalizeJSON(value, fallback)
	if len(normalized) == 0 || !json.Valid(normalized) {
		return ErrInvalidJSON
	}
	return nil
}

func toOutput(secret domain.Secret, encryptor Encryptor) (SecretOutput, error) {
	metadata, err := encryptor.Decrypt(secret.Metadata, secret.MetadataNonce)
	if err != nil {
		return SecretOutput{}, fmt.Errorf("decrypt metadata: %w", err)
	}
	var payload json.RawMessage
	if domain.IsStructuredType(secret.Type) {
		payloadBytes, err := encryptor.Decrypt(secret.Payload, secret.PayloadNonce)
		if err != nil {
			return SecretOutput{}, fmt.Errorf("decrypt payload: %w", err)
		}
		payload = json.RawMessage(payloadBytes)
	}
	return SecretOutput{
		ID:        secret.ID,
		UserID:    secret.UserID,
		Type:      secret.Type,
		Name:      secret.Name,
		Metadata:  json.RawMessage(metadata),
		Payload:   payload,
		Version:   secret.Version,
		CreatedAt: secret.CreatedAt,
		UpdatedAt: secret.UpdatedAt,
	}, nil
}

func toBlobOutput(blob domain.Blob) BlobOutput {
	return BlobOutput{
		ID:             blob.ID,
		OriginalName:   blob.OriginalName,
		ContentType:    blob.ContentType,
		Size:           blob.Size,
		ChecksumSHA256: blob.ChecksumSHA256,
	}
}

func toListItem(secret domain.Secret, encryptor Encryptor) (SecretListItem, error) {
	metadata, err := encryptor.Decrypt(secret.Metadata, secret.MetadataNonce)
	if err != nil {
		return SecretListItem{}, fmt.Errorf("decrypt metadata: %w", err)
	}
	return SecretListItem{
		ID:        secret.ID,
		Type:      secret.Type,
		Name:      secret.Name,
		Metadata:  json.RawMessage(metadata),
		Version:   secret.Version,
		CreatedAt: secret.CreatedAt,
		UpdatedAt: secret.UpdatedAt,
	}, nil
}

func encryptJSON(encryptor Encryptor, value json.RawMessage, fallback []byte) ([]byte, []byte, error) {
	encrypted, nonce, err := encryptor.Encrypt(normalizeJSON(value, fallback))
	if err != nil {
		return nil, nil, err
	}
	return encrypted, nonce, nil
}
