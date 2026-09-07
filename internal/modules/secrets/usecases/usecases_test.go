package usecases

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
)

type repositoryMock struct {
	secret        domain.Secret
	secrets       []domain.Secret
	err           error
	blob          domain.Blob
	blobErr       error
	saved         domain.Secret
	savedBlob     domain.Blob
	updated       domain.Secret
	expected      int
	deleted       bool
	deletedBlobID uuid.UUID
	cleanupBlobs  []domain.Blob
	storageBlobID uuid.UUID
}

func (m *repositoryMock) Save(_ context.Context, secret domain.Secret) error {
	m.saved = secret
	return m.err
}

func (m *repositoryMock) Load(_ context.Context, _ uuid.UUID, _ uuid.UUID) (domain.Secret, error) {
	return m.secret, m.err
}

func (m *repositoryMock) List(_ context.Context, _ uuid.UUID) ([]domain.Secret, error) {
	return m.secrets, m.err
}

func (m *repositoryMock) Update(_ context.Context, secret domain.Secret, expectedVersion int) error {
	m.updated = secret
	m.expected = expectedVersion
	return m.err
}

func (m *repositoryMock) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	m.deleted = true
	return m.err
}

func (m *repositoryMock) SaveBlob(_ context.Context, blob domain.Blob) error {
	m.savedBlob = blob
	return m.blobErr
}

func (m *repositoryMock) LoadBlobBySecret(_ context.Context, _ uuid.UUID, _ uuid.UUID) (domain.Blob, error) {
	return m.blob, m.blobErr
}

func (m *repositoryMock) MarkBlobDeleted(_ context.Context, _ uuid.UUID, blobID uuid.UUID) error {
	m.deletedBlobID = blobID
	return m.blobErr
}

func (m *repositoryMock) ListBlobsForCleanup(_ context.Context, _ time.Time, _ int) ([]domain.Blob, error) {
	return m.cleanupBlobs, m.blobErr
}

func (m *repositoryMock) MarkBlobStorageDeleted(_ context.Context, _ uuid.UUID, blobID uuid.UUID) error {
	m.storageBlobID = blobID
	return m.blobErr
}

type encryptorMock struct {
	err error
}

func (m *encryptorMock) Encrypt(plaintext []byte) ([]byte, []byte, error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	return append([]byte("encrypted:"), plaintext...), []byte("nonce"), nil
}

func (m *encryptorMock) Decrypt(ciphertext []byte, _ []byte) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []byte(string(ciphertext)[len("encrypted:"):]), nil
}

func (m *encryptorMock) EncryptStream(src io.Reader, dst io.Writer) (int64, string, error) {
	if m.err != nil {
		return 0, "", m.err
	}
	data, err := io.ReadAll(src)
	if err != nil {
		return 0, "", err
	}
	if _, err = dst.Write(append([]byte("stream:"), data...)); err != nil {
		return 0, "", err
	}
	return int64(len(data)), "checksum", nil
}

func (m *encryptorMock) DecryptStream(src io.Reader, dst io.Writer) error {
	if m.err != nil {
		return m.err
	}
	data, err := io.ReadAll(src)
	if err != nil {
		return err
	}
	_, err = dst.Write([]byte(strings.TrimPrefix(string(data), "stream:")))
	return err
}

type storageMock struct {
	content bytes.Buffer
	err     error
	deleted string
}

func (m *storageMock) Save(_ context.Context, _ string, reader io.Reader) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	_, err := io.Copy(&m.content, reader)
	return "storage/path", err
}

func (m *storageMock) Open(_ context.Context, _ string) (io.ReadCloser, error) {
	if m.err != nil {
		return nil, m.err
	}
	return io.NopCloser(bytes.NewReader(m.content.Bytes())), nil
}

func (m *storageMock) Delete(_ context.Context, storageName string) error {
	m.deleted = storageName
	return m.err
}

func TestUseCases(t *testing.T) {
	t.Run(
		"CreateUseCase", func(t *testing.T) {
			t.Run(
				"Должен создать secret", func(t *testing.T) {
					repository := &repositoryMock{}
					useCase, err := NewCreateUseCase(repository, &encryptorMock{})
					require.NoError(t, err)
					userID := uuid.New()

					output, err := useCase.Execute(
						context.Background(),
						SecretInput{
							UserID:   userID,
							Type:     domain.SecretTypeCredentials,
							Name:     "github",
							Metadata: json.RawMessage(`{"site":"github"}`),
							Payload:  json.RawMessage(`{"login":"igor","password":"secret"}`),
						},
					)

					require.NoError(t, err)
					assert.Equal(t, userID, output.UserID)
					assert.Equal(t, "github", output.Name)
					assert.JSONEq(t, `{"site":"github"}`, string(output.Metadata))
					assert.JSONEq(t, `{"login":"igor","password":"secret"}`, string(output.Payload))
					assert.Equal(t, 1, repository.saved.Version)
					assert.NotEmpty(t, repository.saved.MetadataNonce)
				},
			)

			t.Run(
				"Должен вернуть ошибку при неподдержанном типе", func(t *testing.T) {
					useCase, err := NewCreateUseCase(&repositoryMock{}, &encryptorMock{})
					require.NoError(t, err)

					_, err = useCase.Execute(
						context.Background(),
						SecretInput{Type: domain.SecretType("text"), Payload: json.RawMessage(`{}`)},
					)

					require.Error(t, err)
					assert.ErrorIs(t, err, ErrInvalidSecretType)
				},
			)

			t.Run(
				"Должен вернуть ошибку при невалидном JSON", func(t *testing.T) {
					useCase, err := NewCreateUseCase(&repositoryMock{}, &encryptorMock{})
					require.NoError(t, err)

					_, err = useCase.Execute(
						context.Background(),
						SecretInput{Type: domain.SecretTypeCard, Payload: json.RawMessage(`{`)},
					)

					require.Error(t, err)
					assert.ErrorIs(t, err, ErrInvalidJSON)
				},
			)
		},
	)

	t.Run(
		"BlobUseCases", func(t *testing.T) {
			t.Run(
				"Должен создать blob secret", func(t *testing.T) {
					repository := &repositoryMock{}
					storage := &storageMock{}
					useCase, err := NewCreateBlobUseCase(repository, &encryptorMock{}, storage)
					require.NoError(t, err)
					userID := uuid.New()

					output, err := useCase.Execute(
						context.Background(),
						BlobSecretInput{
							UserID:       userID,
							Type:         domain.SecretTypeText,
							Name:         "war-and-peace",
							Metadata:     json.RawMessage(`{"kind":"book"}`),
							OriginalName: "book.txt",
							ContentType:  "text/plain",
							Content:      strings.NewReader("content"),
						},
					)

					require.NoError(t, err)
					assert.Equal(t, domain.SecretTypeText, output.Type)
					require.NotNil(t, output.Blob)
					assert.Equal(t, "book.txt", output.Blob.OriginalName)
					assert.Equal(t, int64(7), output.Blob.Size)
					assert.NotNil(t, repository.saved.BlobID)
					assert.Equal(t, []byte("stream:content"), storage.content.Bytes())
				},
			)

			t.Run(
				"Должен вернуть stream содержимого blob", func(t *testing.T) {
					blob := domain.Blob{
						ID:             uuid.New(),
						UserID:         uuid.New(),
						OriginalName:   "book.txt",
						StorageName:    "book.gpk",
						ContentType:    "text/plain",
						Size:           7,
						ChecksumSHA256: "checksum",
					}
					storage := &storageMock{content: *bytes.NewBufferString("stream:content")}
					useCase, err := NewGetBlobContentUseCase(&repositoryMock{blob: blob}, &encryptorMock{}, storage)
					require.NoError(t, err)

					output, err := useCase.Execute(context.Background(), blob.UserID, uuid.New())
					require.NoError(t, err)
					defer output.Content.Close()

					data, err := io.ReadAll(output.Content)
					require.NoError(t, err)
					assert.Equal(t, "content", string(data))
					assert.Equal(t, "book.txt", output.Blob.OriginalName)
				},
			)

			t.Run(
				"Должен заменить содержимое blob secret", func(t *testing.T) {
					userID := uuid.New()
					secretID := uuid.New()
					oldBlobID := uuid.New()
					now := time.Now().UTC()
					secret := domain.Secret{
						ID:            secretID,
						UserID:        userID,
						Type:          domain.SecretTypeText,
						Name:          "book",
						Metadata:      []byte(`encrypted:{}`),
						MetadataNonce: []byte("nonce"),
						Payload:       []byte(`encrypted:{}`),
						PayloadNonce:  []byte("nonce"),
						BlobID:        &oldBlobID,
						Version:       2,
						CreatedAt:     now,
						UpdatedAt:     now,
					}
					oldBlob := domain.Blob{
						ID:             oldBlobID,
						UserID:         userID,
						OriginalName:   "old.txt",
						StorageName:    "old.gpk",
						ContentType:    "text/plain",
						Size:           3,
						ChecksumSHA256: "old-checksum",
						CreatedAt:      now,
					}
					repository := &repositoryMock{secret: secret, blob: oldBlob}
					storage := &storageMock{}
					useCase, err := NewUpdateBlobContentUseCase(repository, &encryptorMock{}, storage)
					require.NoError(t, err)

					output, err := useCase.Execute(
						context.Background(),
						BlobContentInput{
							UserID:          userID,
							ID:              secretID,
							ExpectedVersion: 2,
							OriginalName:    "new.txt",
							ContentType:     "text/plain",
							Content:         strings.NewReader("new"),
						},
					)

					require.NoError(t, err)
					assert.Equal(t, 3, output.Version)
					require.NotNil(t, output.Blob)
					assert.Equal(t, "new.txt", output.Blob.OriginalName)
					require.NotNil(t, repository.updated.BlobID)
					assert.NotEqual(t, oldBlobID, *repository.updated.BlobID)
					assert.Equal(t, oldBlobID, repository.deletedBlobID)
					assert.Equal(t, 2, repository.expected)
					assert.Equal(t, []byte("stream:new"), storage.content.Bytes())
				},
			)

			t.Run(
				"Должен удалить физические файлы soft-deleted blob", func(t *testing.T) {
					blob := domain.Blob{
						ID:          uuid.New(),
						UserID:      uuid.New(),
						StorageName: "old.gpk",
					}
					repository := &repositoryMock{cleanupBlobs: []domain.Blob{blob}}
					storage := &storageMock{}
					useCase, err := NewCleanupBlobsUseCase(repository, storage)
					require.NoError(t, err)

					output, err := useCase.Execute(context.Background(), CleanupBlobsInput{})

					require.NoError(t, err)
					assert.Equal(t, 1, output.Deleted)
					assert.Equal(t, "old.gpk", storage.deleted)
					assert.Equal(t, blob.ID, repository.storageBlobID)
				},
			)
		},
	)

	t.Run(
		"Get/List/Update/Delete", func(t *testing.T) {
			userID := uuid.New()
			secretID := uuid.New()
			now := time.Now().UTC()
			secret := domain.Secret{
				ID:            secretID,
				UserID:        userID,
				Type:          domain.SecretTypeCredentials,
				Name:          "github",
				Metadata:      []byte(`encrypted:{"site":"github"}`),
				MetadataNonce: []byte("nonce"),
				Payload:       []byte(`encrypted:{"login":"igor"}`),
				PayloadNonce:  []byte("nonce"),
				Version:       2,
				CreatedAt:     now,
				UpdatedAt:     now,
			}

			t.Run(
				"Должен получить secret", func(t *testing.T) {
					useCase, err := NewGetUseCase(&repositoryMock{secret: secret}, &encryptorMock{})
					require.NoError(t, err)

					output, err := useCase.Execute(context.Background(), userID, secretID)

					require.NoError(t, err)
					assert.Equal(t, secretID, output.ID)
					assert.JSONEq(t, `{"login":"igor"}`, string(output.Payload))
				},
			)

			t.Run(
				"Должен вернуть список", func(t *testing.T) {
					useCase, err := NewListUseCase(&repositoryMock{secrets: []domain.Secret{secret}}, &encryptorMock{})
					require.NoError(t, err)

					output, err := useCase.Execute(context.Background(), userID)

					require.NoError(t, err)
					require.Len(t, output, 1)
					assert.Equal(t, secretID, output[0].ID)
					assert.JSONEq(t, `{"site":"github"}`, string(output[0].Metadata))
				},
			)

			t.Run(
				"Должен обновить secret", func(t *testing.T) {
					repository := &repositoryMock{secret: secret}
					useCase, err := NewUpdateUseCase(repository, &encryptorMock{})
					require.NoError(t, err)

					output, err := useCase.Execute(
						context.Background(),
						UpdateSecretInput{
							UserID:   userID,
							ID:       secretID,
							Type:     domain.SecretTypeCard,
							Name:     "card",
							Metadata: json.RawMessage(`{}`),
							Payload: json.RawMessage(
								`{"number":"1234","holder":"IGOR","expires_at":"12/30","cvv":"123"}`,
							),
							ExpectedVersion: 2,
						},
					)

					require.NoError(t, err)
					assert.Equal(t, 3, output.Version)
					assert.Equal(t, domain.SecretTypeCard, repository.updated.Type)
					assert.Equal(t, 2, repository.expected)
				},
			)

			t.Run(
				"Должен вернуть conflict при неактуальной версии", func(t *testing.T) {
					repository := &repositoryMock{secret: secret}
					useCase, err := NewUpdateUseCase(repository, &encryptorMock{})
					require.NoError(t, err)

					_, err = useCase.Execute(
						context.Background(),
						UpdateSecretInput{
							UserID:   userID,
							ID:       secretID,
							Type:     domain.SecretTypeCard,
							Name:     "card",
							Metadata: json.RawMessage(`{}`),
							Payload: json.RawMessage(
								`{"number":"1234","holder":"IGOR","expires_at":"12/30","cvv":"123"}`,
							),
							ExpectedVersion: 1,
						},
					)

					require.Error(t, err)
					assert.ErrorIs(t, err, ErrSecretVersionConflict)
				},
			)

			t.Run(
				"Должен удалить secret", func(t *testing.T) {
					repository := &repositoryMock{}
					useCase, err := NewDeleteUseCase(repository)
					require.NoError(t, err)

					err = useCase.Execute(context.Background(), userID, secretID)

					require.NoError(t, err)
					assert.True(t, repository.deleted)
				},
			)
		},
	)
}
