package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/igor/gophkeeper/internal/modules/secrets/domain"
)

// SyncUseCase возвращает изменения секретов пользователя для синхронизации клиента.
type SyncUseCase struct {
	repository Repository
	encryptor  Encryptor
}

// NewSyncUseCase создает SyncUseCase.
func NewSyncUseCase(repository Repository, encryptor Encryptor) (*SyncUseCase, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: repository", ErrEmptyDependency)
	}
	if encryptor == nil {
		return nil, fmt.Errorf("%w: encryptor", ErrEmptyDependency)
	}
	return &SyncUseCase{repository: repository, encryptor: encryptor}, nil
}

// Execute возвращает полный активный snapshot или изменения после since.
func (u *SyncUseCase) Execute(ctx context.Context, input SyncInput) (SyncOutput, error) {
	var (
		secrets []domain.Secret
		err     error
	)
	if input.Since == nil {
		secrets, err = u.repository.List(ctx, input.UserID)
	} else {
		secrets, err = u.repository.ListChanged(ctx, input.UserID, input.Since.UTC())
	}
	if err != nil {
		return SyncOutput{}, fmt.Errorf("list sync secrets: %w", err)
	}

	output := SyncOutput{
		ServerTime: time.Now().UTC(),
		Secrets:    make([]SecretOutput, 0),
		Deleted:    make([]DeletedSecretOutput, 0),
	}
	for _, item := range secrets {
		if item.DeletedAt != nil {
			output.Deleted = append(
				output.Deleted,
				DeletedSecretOutput{
					ID:        item.ID,
					Version:   item.Version,
					DeletedAt: item.DeletedAt.UTC(),
				},
			)
			continue
		}
		secret, err := toOutput(item, u.encryptor)
		if err != nil {
			return SyncOutput{}, err
		}
		if item.BlobID != nil {
			blob, err := u.repository.LoadBlobBySecret(ctx, input.UserID, item.ID)
			if err != nil {
				return SyncOutput{}, fmt.Errorf("load blob: %w", err)
			}
			secret.Blob = ptrBlobOutput(toBlobOutput(blob))
		}
		output.Secrets = append(output.Secrets, secret)
	}
	return output, nil
}
