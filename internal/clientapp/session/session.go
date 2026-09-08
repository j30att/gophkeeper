// Package session хранит локальную пользовательскую сессию CLI-клиента.
package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotFound возвращается, когда локальной сессии еще нет.
var ErrNotFound = errors.New("session not found")

// Session описывает сохраненную пользовательскую сессию.
type Session struct {
	Token string `json:"token"`
}

// Store сохраняет и загружает локальную сессию.
type Store struct {
	path string
}

// NewStore создает Store.
func NewStore(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("empty session path")
	}
	return &Store{path: path}, nil
}

// Save сохраняет сессию на диск.
func (s *Store) Save(session Session) error {
	if strings.TrimSpace(session.Token) == "" {
		return fmt.Errorf("empty token")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}
	payload, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	if err = os.WriteFile(s.path, payload, 0o600); err != nil {
		return fmt.Errorf("write session: %w", err)
	}
	return nil
}

// Load загружает сессию с диска.
func (s *Store) Load() (Session, error) {
	payload, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Session{}, ErrNotFound
		}
		return Session{}, fmt.Errorf("read session: %w", err)
	}
	var session Session
	if err = json.Unmarshal(payload, &session); err != nil {
		return Session{}, fmt.Errorf("unmarshal session: %w", err)
	}
	if strings.TrimSpace(session.Token) == "" {
		return Session{}, ErrNotFound
	}
	return session, nil
}
