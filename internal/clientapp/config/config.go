// Package config содержит конфигурацию CLI-клиента GophKeeper.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultServerURL = "http://127.0.0.1:8080"

// Config описывает настройки CLI-клиента.
type Config struct {
	ServerURL   string
	SessionPath string
	SecretsPath string
}

// Load загружает конфигурацию клиента из flags и environment variables.
func Load(serverURL string) (Config, error) {
	if strings.TrimSpace(serverURL) == "" {
		serverURL = os.Getenv("GOPHKEEPER_SERVER_URL")
	}
	if strings.TrimSpace(serverURL) == "" {
		serverURL = defaultServerURL
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("get user home dir: %w", err)
	}
	return Config{
		ServerURL:   strings.TrimRight(serverURL, "/"),
		SessionPath: filepath.Join(homeDir, ".gophkeeper", "session.json"),
		SecretsPath: filepath.Join(homeDir, ".gophkeeper", "secrets.json"),
	}, nil
}
