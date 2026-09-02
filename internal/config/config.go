// Package config содержит конфигурацию приложения.
package config

import (
	"os"
	"time"
)

const (
	defaultServerAddress = ":8080"
	defaultPostgresDSN   = "postgres://gophkeeper:gophkeeper@localhost:5432/gophkeeper?sslmode=disable"
	defaultJWTSecret     = "dev-secret-change-me"
	defaultTokenTTL      = 24 * time.Hour
)

// Config описывает конфигурацию сервера GophKeeper.
type Config struct {
	Server   Server
	Postgres Postgres
	Auth     Auth
}

// Server описывает HTTP-конфигурацию сервера.
type Server struct {
	Address string
}

// Postgres описывает конфигурацию подключения к Postgres.
type Postgres struct {
	DSN string
}

// Auth описывает конфигурацию авторизации.
type Auth struct {
	JWTSecret      string
	AccessTokenTTL time.Duration
}

// Load загружает конфигурацию из environment variables с dev-значениями по умолчанию.
func Load() Config {
	return Config{
		Server: Server{
			Address: stringFromEnv("GOPHKEEPER_SERVER_ADDRESS", defaultServerAddress),
		},
		Postgres: Postgres{
			DSN: stringFromEnv("GOPHKEEPER_POSTGRES_DSN", defaultPostgresDSN),
		},
		Auth: Auth{
			JWTSecret:      stringFromEnv("GOPHKEEPER_JWT_SECRET", defaultJWTSecret),
			AccessTokenTTL: durationFromEnv("GOPHKEEPER_ACCESS_TOKEN_TTL", defaultTokenTTL),
		},
	}
}

func stringFromEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func durationFromEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}
