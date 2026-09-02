// Package memory содержит in-memory auth repositories для ранней разработки и тестов.
package memory

import (
	"context"
	"sync"

	"github.com/igor/gophkeeper/internal/modules/auth/domain"
	"github.com/igor/gophkeeper/internal/modules/auth/usecases"
)

// UserRepository хранит пользователей в памяти.
type UserRepository struct {
	mu      sync.RWMutex
	byLogin map[string]domain.User
}

// NewUserRepository создает UserRepository.
func NewUserRepository() *UserRepository {
	return &UserRepository{byLogin: make(map[string]domain.User)}
}

// Save сохраняет пользователя по login.
func (r *UserRepository) Save(_ context.Context, user domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byLogin[user.Login] = user
	return nil
}

// Load возвращает пользователя по login.
func (r *UserRepository) Load(_ context.Context, login string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.byLogin[login]
	if !ok {
		return domain.User{}, usecases.ErrUserNotFound
	}
	return user, nil
}
