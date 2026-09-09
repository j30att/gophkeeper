// Package cache хранит локальный cache секретов CLI-клиента.
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/igor/gophkeeper/internal/clientapp/api"
)

// ErrNotFound возвращается, когда локального cache еще нет.
var ErrNotFound = errors.New("cache not found")

// Cache описывает локально сохраненный snapshot секретов.
type Cache struct {
	LastSyncAt *time.Time   `json:"last_sync_at,omitempty"`
	Secrets    []api.Secret `json:"secrets"`
}

// Store сохраняет и загружает локальный cache секретов.
type Store struct {
	path string
}

// NewStore создает Store.
func NewStore(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("empty cache path")
	}
	return &Store{path: path}, nil
}

// Save сохраняет cache на диск.
func (s *Store) Save(cache Cache) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	payload, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}
	if err = os.WriteFile(s.path, payload, 0o600); err != nil {
		return fmt.Errorf("write cache: %w", err)
	}
	return nil
}

// Load загружает cache с диска.
func (s *Store) Load() (Cache, error) {
	payload, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Cache{}, ErrNotFound
		}
		return Cache{}, fmt.Errorf("read cache: %w", err)
	}
	var cache Cache
	if err = json.Unmarshal(payload, &cache); err != nil {
		return Cache{}, fmt.Errorf("unmarshal cache: %w", err)
	}
	return cache, nil
}

// Delete удаляет cache с диска.
func (s *Store) Delete() error {
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete cache: %w", err)
	}
	return nil
}
