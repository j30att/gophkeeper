// Package filesystem содержит локальное файловое blob-хранилище.
package filesystem

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// BlobStorage сохраняет blob-файлы на локальной файловой системе.
type BlobStorage struct {
	root string
}

// NewBlobStorage создает BlobStorage.
func NewBlobStorage(root string) (*BlobStorage, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("empty blob storage path")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create blob storage dir: %w", err)
	}
	return &BlobStorage{root: root}, nil
}

// Save сохраняет содержимое reader под storageName.
func (s *BlobStorage) Save(ctx context.Context, storageName string, reader io.Reader) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := s.path(storageName)
	if err != nil {
		return "", err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create blob file: %w", err)
	}
	defer file.Close()

	if _, err = io.Copy(file, reader); err != nil {
		return "", fmt.Errorf("write blob file: %w", err)
	}
	return path, nil
}

// Open открывает файл blob на чтение.
func (s *BlobStorage) Open(ctx context.Context, storageName string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := s.path(storageName)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open blob file: %w", err)
	}
	return file, nil
}

func (s *BlobStorage) path(storageName string) (string, error) {
	if strings.TrimSpace(storageName) == "" {
		return "", fmt.Errorf("empty storage name")
	}
	cleanName := filepath.Clean(storageName)
	if cleanName != filepath.Base(cleanName) {
		return "", fmt.Errorf("invalid storage name")
	}
	return filepath.Join(s.root, cleanName), nil
}
