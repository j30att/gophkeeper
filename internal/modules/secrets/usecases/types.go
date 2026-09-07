package usecases

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
)

// Repository сохраняет и загружает JSON-секреты.
type Repository interface {
	Save(ctx context.Context, secret domain.Secret) error
	Load(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (domain.Secret, error)
	List(ctx context.Context, userID uuid.UUID) ([]domain.Secret, error)
	Update(ctx context.Context, secret domain.Secret, expectedVersion int) error
	Delete(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) error
	SaveBlob(ctx context.Context, blob domain.Blob) error
	LoadBlobBySecret(ctx context.Context, userID uuid.UUID, secretID uuid.UUID) (domain.Blob, error)
	MarkBlobDeleted(ctx context.Context, userID uuid.UUID, blobID uuid.UUID) error
	ListBlobsForCleanup(ctx context.Context, before time.Time, limit int) ([]domain.Blob, error)
	MarkBlobStorageDeleted(ctx context.Context, userID uuid.UUID, blobID uuid.UUID) error
}

// Encryptor шифрует и расшифровывает приватные JSON-поля.
type Encryptor interface {
	Encrypt(plaintext []byte) ([]byte, []byte, error)
	Decrypt(ciphertext []byte, nonce []byte) ([]byte, error)
	EncryptStream(src io.Reader, dst io.Writer) (int64, string, error)
	DecryptStream(src io.Reader, dst io.Writer) error
}

// BlobStorage сохраняет и открывает зашифрованные blob-файлы.
type BlobStorage interface {
	Save(ctx context.Context, storageName string, reader io.Reader) (string, error)
	Open(ctx context.Context, storageName string) (io.ReadCloser, error)
	Delete(ctx context.Context, storageName string) error
}

// SecretInput содержит данные JSON-секрета.
type SecretInput struct {
	UserID   uuid.UUID
	Type     domain.SecretType
	Name     string
	Metadata json.RawMessage
	Payload  json.RawMessage
}

// BlobContentInput содержит данные замены содержимого blob-секрета.
type BlobContentInput struct {
	UserID          uuid.UUID
	ID              uuid.UUID
	ExpectedVersion int
	OriginalName    string
	ContentType     string
	Content         io.Reader
}

// BlobSecretInput содержит данные создания blob-секрета.
type BlobSecretInput struct {
	UserID       uuid.UUID
	Type         domain.SecretType
	Name         string
	Metadata     json.RawMessage
	OriginalName string
	ContentType  string
	Content      io.Reader
}

// UpdateSecretInput содержит данные обновления JSON-секрета.
type UpdateSecretInput struct {
	UserID          uuid.UUID
	ID              uuid.UUID
	Type            domain.SecretType
	Name            string
	Metadata        json.RawMessage
	Payload         json.RawMessage
	ExpectedVersion int
}

// SecretOutput содержит расшифрованный JSON-секрет.
type SecretOutput struct {
	ID        uuid.UUID         `json:"id"`
	UserID    uuid.UUID         `json:"user_id"`
	Type      domain.SecretType `json:"type"`
	Name      string            `json:"name"`
	Metadata  json.RawMessage   `json:"metadata"`
	Payload   json.RawMessage   `json:"payload,omitempty"`
	Blob      *BlobOutput       `json:"blob,omitempty"`
	Version   int               `json:"version"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// BlobOutput содержит пользовательскую мету blob-файла.
type BlobOutput struct {
	ID             uuid.UUID `json:"id"`
	OriginalName   string    `json:"original_name"`
	ContentType    string    `json:"content_type"`
	Size           int64     `json:"size"`
	ChecksumSHA256 string    `json:"checksum_sha256"`
}

// BlobContentOutput содержит stream с расшифрованным blob-файлом.
type BlobContentOutput struct {
	Blob    BlobOutput
	Content io.ReadCloser
}

// SecretListItem содержит расшифрованный элемент списка JSON-секретов.
type SecretListItem struct {
	ID        uuid.UUID         `json:"id"`
	Type      domain.SecretType `json:"type"`
	Name      string            `json:"name"`
	Metadata  json.RawMessage   `json:"metadata"`
	Blob      *BlobOutput       `json:"blob,omitempty"`
	Version   int               `json:"version"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}
