// Package postgres содержит Postgres repositories auth-модуля.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/igor/gophkeeper/internal/modules/auth/domain"
	"github.com/igor/gophkeeper/internal/modules/auth/usecases"
)

const (
	uniqueViolationCode = "23505"

	saveUserQuery = `
INSERT INTO users (id, login, password_hash, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5)`

	loadUserQuery = `
SELECT id, login, password_hash, created_at, updated_at
FROM users
WHERE login = $1`
)

// Pool выполняет SQL-запросы к Postgres.
type Pool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// UserRepository хранит пользователей в Postgres.
type UserRepository struct {
	pool Pool
}

// NewUserRepository создает UserRepository.
func NewUserRepository(pool Pool) (*UserRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("%w: pool", usecases.ErrEmptyDependency)
	}
	return &UserRepository{pool: pool}, nil
}

// Save сохраняет пользователя.
func (r *UserRepository) Save(ctx context.Context, user domain.User) error {
	_, err := r.pool.Exec(
		ctx,
		saveUserQuery,
		user.ID,
		user.Login,
		user.PasswordHash,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolationCode {
			return usecases.ErrUserAlreadyExists
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// Load загружает пользователя по login.
func (r *UserRepository) Load(ctx context.Context, login string) (domain.User, error) {
	var user domain.User
	err := r.pool.QueryRow(ctx, loadUserQuery, login).Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, usecases.ErrUserNotFound
		}
		return domain.User{}, fmt.Errorf("select user by login: %w", err)
	}
	user.CreatedAt = user.CreatedAt.UTC()
	user.UpdatedAt = user.UpdatedAt.UTC()
	return user, nil
}
