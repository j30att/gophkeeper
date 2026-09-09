// Package domain содержит доменные модели auth-модуля.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// User описывает учетную запись владельца приватных данных GophKeeper.
type User struct {
	ID           uuid.UUID
	Login        string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
