package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/igor/gophkeeper/internal/modules/auth/domain"
	"github.com/igor/gophkeeper/internal/modules/auth/usecases"
)

type poolMock struct {
	execSQL  string
	execArgs []any
	execErr  error
	row      pgx.Row
	querySQL string
	queryArg []any
}

func (m *poolMock) Exec(_ context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	m.execSQL = sql
	m.execArgs = arguments
	return pgconn.CommandTag{}, m.execErr
}

func (m *poolMock) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	m.querySQL = sql
	m.queryArg = args
	return m.row
}

type rowMock struct {
	user domain.User
	err  error
}

func (r *rowMock) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*uuid.UUID)) = r.user.ID
	*(dest[1].(*string)) = r.user.Login
	*(dest[2].(*string)) = r.user.PasswordHash
	*(dest[3].(*time.Time)) = r.user.CreatedAt
	*(dest[4].(*time.Time)) = r.user.UpdatedAt
	return nil
}

func TestUserRepository(t *testing.T) {
	t.Run(
		"Тест создания repository", func(t *testing.T) {
			t.Run(
				"Должен создать repository без ошибок", func(t *testing.T) {
					pool := &poolMock{}

					repository, err := NewUserRepository(pool)

					require.NoError(t, err)
					assert.Equal(t, pool, repository.pool)
				},
			)

			t.Run(
				"Ошибка, отсутствует pool", func(t *testing.T) {
					_, err := NewUserRepository(nil)

					require.Error(t, err)
					assert.ErrorIs(t, err, usecases.ErrEmptyDependency)
				},
			)
		},
	)

	t.Run(
		"Тест метода Save", func(t *testing.T) {
			t.Run(
				"Должен сохранить пользователя", func(t *testing.T) {
					pool := &poolMock{}
					repository, err := NewUserRepository(pool)
					require.NoError(t, err)
					user := domain.User{
						ID:           uuid.New(),
						Login:        "igor",
						PasswordHash: "hash",
						CreatedAt:    time.Now().UTC(),
						UpdatedAt:    time.Now().UTC(),
					}

					err = repository.Save(context.Background(), user)

					require.NoError(t, err)
					assert.Equal(t, saveUserQuery, pool.execSQL)
					require.Len(t, pool.execArgs, 5)
					assert.Equal(t, user.ID, pool.execArgs[0])
					assert.Equal(t, user.Login, pool.execArgs[1])
					assert.Equal(t, user.PasswordHash, pool.execArgs[2])
				},
			)

			t.Run(
				"Должен вернуть ошибку insert", func(t *testing.T) {
					pool := &poolMock{execErr: assert.AnError}
					repository, err := NewUserRepository(pool)
					require.NoError(t, err)

					err = repository.Save(context.Background(), domain.User{})

					require.Error(t, err)
					assert.ErrorIs(t, err, assert.AnError)
				},
			)

			t.Run(
				"Должен вернуть ErrUserAlreadyExists при unique violation", func(t *testing.T) {
					pool := &poolMock{execErr: &pgconn.PgError{Code: uniqueViolationCode}}
					repository, err := NewUserRepository(pool)
					require.NoError(t, err)

					err = repository.Save(context.Background(), domain.User{})

					require.Error(t, err)
					assert.ErrorIs(t, err, usecases.ErrUserAlreadyExists)
				},
			)
		},
	)

	t.Run(
		"Тест метода Load", func(t *testing.T) {
			t.Run(
				"Должен загрузить пользователя", func(t *testing.T) {
					user := domain.User{
						ID:           uuid.New(),
						Login:        "igor",
						PasswordHash: "hash",
						CreatedAt:    time.Now(),
						UpdatedAt:    time.Now(),
					}
					pool := &poolMock{row: &rowMock{user: user}}
					repository, err := NewUserRepository(pool)
					require.NoError(t, err)

					loaded, err := repository.Load(context.Background(), "igor")

					require.NoError(t, err)
					assert.Equal(t, loadUserQuery, pool.querySQL)
					assert.Equal(t, []any{"igor"}, pool.queryArg)
					assert.Equal(t, user.ID, loaded.ID)
					assert.Equal(t, user.Login, loaded.Login)
					assert.Equal(t, user.PasswordHash, loaded.PasswordHash)
				},
			)

			t.Run(
				"Должен вернуть ErrUserNotFound", func(t *testing.T) {
					pool := &poolMock{row: &rowMock{err: pgx.ErrNoRows}}
					repository, err := NewUserRepository(pool)
					require.NoError(t, err)

					_, err = repository.Load(context.Background(), "missing")

					require.Error(t, err)
					assert.ErrorIs(t, err, usecases.ErrUserNotFound)
				},
			)

			t.Run(
				"Должен вернуть ошибку select", func(t *testing.T) {
					pool := &poolMock{row: &rowMock{err: assert.AnError}}
					repository, err := NewUserRepository(pool)
					require.NoError(t, err)

					_, err = repository.Load(context.Background(), "igor")

					require.Error(t, err)
					assert.ErrorIs(t, err, assert.AnError)
				},
			)
		},
	)
}
