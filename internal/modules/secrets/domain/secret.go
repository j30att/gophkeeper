// Package domain содержит доменные модели secrets-модуля.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// SecretType описывает тип приватной JSON-записи.
type SecretType string

const (
	// SecretTypeCredentials описывает пару login/password.
	SecretTypeCredentials SecretType = "credentials"
	// SecretTypeCard описывает банковскую карту.
	SecretTypeCard SecretType = "card"
	// SecretTypeText описывает произвольный текст, который хранится как blob-файл.
	SecretTypeText SecretType = "text"
	// SecretTypeBinary описывает произвольные бинарные данные, которые хранятся как blob-файл.
	SecretTypeBinary SecretType = "binary"
)

// Secret содержит зашифрованную JSON-запись пользователя.
type Secret struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Type          SecretType
	Name          string
	Metadata      []byte
	MetadataNonce []byte
	Payload       []byte
	PayloadNonce  []byte
	BlobID        *uuid.UUID
	Version       int
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

// Blob содержит метаданные физического файла с зашифрованным содержимым.
type Blob struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	OriginalName   string
	StorageName    string
	StoragePath    string
	ContentType    string
	Size           int64
	ChecksumSHA256 string
	CreatedAt      time.Time
	DeletedAt      *time.Time
}

// IsStructuredType возвращает true для JSON-типов, которые хранятся в payload.
func IsStructuredType(secretType SecretType) bool {
	return secretType == SecretTypeCredentials || secretType == SecretTypeCard
}

// IsBlobType возвращает true для типов, содержимое которых хранится как файл.
func IsBlobType(secretType SecretType) bool {
	return secretType == SecretTypeText || secretType == SecretTypeBinary
}
