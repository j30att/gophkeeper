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

	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
	"github.com/igor/gophkeeper/internal/modules/secrets/usecases"
)

type poolMock struct {
	execTag  pgconn.CommandTag
	execErr  error
	execSQL  string
	execArgs []any
	row      pgx.Row
	rows     pgx.Rows
	queryErr error
}

func (m *poolMock) Exec(_ context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	m.execSQL = sql
	m.execArgs = arguments
	return m.execTag, m.execErr
}

func (m *poolMock) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return m.rows, m.queryErr
}

func (m *poolMock) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return m.row
}

type rowMock struct {
	secret domain.Secret
	blob   domain.Blob
	isBlob bool
	count  *int
	err    error
}

func (r *rowMock) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if r.count != nil {
		*(dest[0].(*int)) = *r.count
		return nil
	}
	if r.isBlob {
		scanBlobValues(r.blob, dest...)
		return nil
	}
	scanSecretValues(r.secret, dest...)
	return nil
}

type rowsMock struct {
	secrets []domain.Secret
	cursor  int
	err     error
	scanErr error
}

func (r *rowsMock) Next() bool {
	return r.cursor < len(r.secrets)
}

func (r *rowsMock) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	scanSecretValues(r.secrets[r.cursor], dest...)
	r.cursor++
	return nil
}

func (_ *rowsMock) Close()                                       {}
func (r *rowsMock) Err() error                                   { return r.err }
func (_ *rowsMock) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (_ *rowsMock) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (_ *rowsMock) RawValues() [][]byte                          { return nil }
func (_ *rowsMock) Values() ([]any, error)                       { return nil, nil }
func (_ *rowsMock) Conn() *pgx.Conn                              { return nil }

func TestRepository(t *testing.T) {
	t.Run(
		"Должен создать repository", func(t *testing.T) {
			repository, err := NewRepository(&poolMock{})

			require.NoError(t, err)
			assert.NotNil(t, repository)
		},
	)

	t.Run(
		"Должен вернуть ошибку без pool", func(t *testing.T) {
			_, err := NewRepository(nil)

			require.Error(t, err)
			assert.ErrorIs(t, err, usecases.ErrEmptyDependency)
		},
	)

	t.Run(
		"Должен сохранить secret", func(t *testing.T) {
			pool := &poolMock{}
			repository, err := NewRepository(pool)
			require.NoError(t, err)
			secret := testSecret()

			err = repository.Save(context.Background(), secret)

			require.NoError(t, err)
			assert.Equal(t, saveSecretQuery, pool.execSQL)
			require.Len(t, pool.execArgs, 12)
		},
	)

	t.Run(
		"Должен вернуть ошибку save", func(t *testing.T) {
			repository, err := NewRepository(&poolMock{execErr: assert.AnError})
			require.NoError(t, err)

			err = repository.Save(context.Background(), testSecret())

			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
		},
	)

	t.Run(
		"Должен загрузить secret", func(t *testing.T) {
			secret := testSecret()
			repository, err := NewRepository(&poolMock{row: &rowMock{secret: secret}})
			require.NoError(t, err)

			loaded, err := repository.Load(context.Background(), secret.UserID, secret.ID)

			require.NoError(t, err)
			assert.Equal(t, secret.ID, loaded.ID)
			assert.Equal(t, secret.UserID, loaded.UserID)
		},
	)

	t.Run(
		"Должен вернуть not found при отсутствии secret", func(t *testing.T) {
			repository, err := NewRepository(&poolMock{row: &rowMock{err: pgx.ErrNoRows}})
			require.NoError(t, err)

			_, err = repository.Load(context.Background(), uuid.New(), uuid.New())

			require.Error(t, err)
			assert.ErrorIs(t, err, usecases.ErrSecretNotFound)
		},
	)

	t.Run(
		"Должен вернуть список", func(t *testing.T) {
			secret := testSecret()
			repository, err := NewRepository(&poolMock{rows: &rowsMock{secrets: []domain.Secret{secret}}})
			require.NoError(t, err)

			list, err := repository.List(context.Background(), secret.UserID)

			require.NoError(t, err)
			require.Len(t, list, 1)
			assert.Equal(t, secret.ID, list[0].ID)
		},
	)

	t.Run(
		"Должен вернуть ошибку query list", func(t *testing.T) {
			repository, err := NewRepository(&poolMock{queryErr: assert.AnError})
			require.NoError(t, err)

			_, err = repository.List(context.Background(), uuid.New())

			require.Error(t, err)
			assert.ErrorIs(t, err, assert.AnError)
		},
	)

	t.Run(
		"Должен обновить secret", func(t *testing.T) {
			repository, err := NewRepository(&poolMock{execTag: pgconn.NewCommandTag("UPDATE 1")})
			require.NoError(t, err)

			err = repository.Update(context.Background(), testSecret(), 1)

			require.NoError(t, err)
		},
	)

	t.Run(
		"Должен вернуть not found при update без rows", func(t *testing.T) {
			repository, err := NewRepository(&poolMock{execTag: pgconn.NewCommandTag("UPDATE 0")})
			require.NoError(t, err)

			err = repository.Update(context.Background(), testSecret(), 1)

			require.Error(t, err)
			assert.ErrorIs(t, err, usecases.ErrSecretVersionConflict)
		},
	)

	t.Run(
		"Должен удалить secret", func(t *testing.T) {
			secret := testSecret()
			deletedCount := 1
			repository, err := NewRepository(&poolMock{row: &rowMock{count: &deletedCount}})
			require.NoError(t, err)

			err = repository.Delete(context.Background(), secret.UserID, secret.ID)

			require.NoError(t, err)
		},
	)

	t.Run(
		"Должен вернуть not found при delete без rows", func(t *testing.T) {
			deletedCount := 0
			repository, err := NewRepository(&poolMock{row: &rowMock{count: &deletedCount}})
			require.NoError(t, err)

			err = repository.Delete(context.Background(), uuid.New(), uuid.New())

			require.Error(t, err)
			assert.ErrorIs(t, err, usecases.ErrSecretNotFound)
		},
	)

	t.Run(
		"Должен пометить blob удаленным", func(t *testing.T) {
			blob := testBlob()
			repository, err := NewRepository(&poolMock{execTag: pgconn.NewCommandTag("UPDATE 1")})
			require.NoError(t, err)

			err = repository.MarkBlobDeleted(context.Background(), blob.UserID, blob.ID)

			require.NoError(t, err)
		},
	)

	t.Run(
		"Должен сохранить blob", func(t *testing.T) {
			pool := &poolMock{}
			repository, err := NewRepository(pool)
			require.NoError(t, err)

			err = repository.SaveBlob(context.Background(), testBlob())

			require.NoError(t, err)
			assert.Equal(t, saveBlobQuery, pool.execSQL)
			require.Len(t, pool.execArgs, 9)
		},
	)

	t.Run(
		"Должен загрузить blob по secret", func(t *testing.T) {
			blob := testBlob()
			repository, err := NewRepository(&poolMock{row: &rowMock{blob: blob, isBlob: true}})
			require.NoError(t, err)

			loaded, err := repository.LoadBlobBySecret(context.Background(), blob.UserID, uuid.New())

			require.NoError(t, err)
			assert.Equal(t, blob.ID, loaded.ID)
			assert.Equal(t, blob.StorageName, loaded.StorageName)
		},
	)
}

func scanSecretValues(secret domain.Secret, dest ...any) {
	*(dest[0].(*uuid.UUID)) = secret.ID
	*(dest[1].(*uuid.UUID)) = secret.UserID
	*(dest[2].(*domain.SecretType)) = secret.Type
	*(dest[3].(*string)) = secret.Name
	*(dest[4].(*[]byte)) = secret.Metadata
	*(dest[5].(*[]byte)) = secret.MetadataNonce
	*(dest[6].(*[]byte)) = secret.Payload
	*(dest[7].(*[]byte)) = secret.PayloadNonce
	*(dest[8].(**uuid.UUID)) = secret.BlobID
	*(dest[9].(*int)) = secret.Version
	*(dest[10].(*time.Time)) = secret.CreatedAt
	*(dest[11].(*time.Time)) = secret.UpdatedAt
	*(dest[12].(**time.Time)) = secret.DeletedAt
}

func scanBlobValues(blob domain.Blob, dest ...any) {
	*(dest[0].(*uuid.UUID)) = blob.ID
	*(dest[1].(*uuid.UUID)) = blob.UserID
	*(dest[2].(*string)) = blob.OriginalName
	*(dest[3].(*string)) = blob.StorageName
	*(dest[4].(*string)) = blob.StoragePath
	*(dest[5].(*string)) = blob.ContentType
	*(dest[6].(*int64)) = blob.Size
	*(dest[7].(*string)) = blob.ChecksumSHA256
	*(dest[8].(*time.Time)) = blob.CreatedAt
	*(dest[9].(**time.Time)) = blob.DeletedAt
}

func testSecret() domain.Secret {
	now := time.Now().UTC()
	return domain.Secret{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		Type:          domain.SecretTypeCredentials,
		Name:          "github",
		Metadata:      []byte("metadata"),
		MetadataNonce: []byte("metadata-nonce"),
		Payload:       []byte("payload"),
		PayloadNonce:  []byte("payload-nonce"),
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func testBlob() domain.Blob {
	now := time.Now().UTC()
	return domain.Blob{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		OriginalName:   "secret.txt",
		StorageName:    uuid.New().String() + ".gpk",
		StoragePath:    "storage/blobs/secret.gpk",
		ContentType:    "text/plain",
		Size:           6,
		ChecksumSHA256: "checksum",
		CreatedAt:      now,
	}
}
