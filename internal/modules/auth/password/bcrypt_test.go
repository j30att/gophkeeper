package password

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBcryptHasher(t *testing.T) {
	t.Run(
		"Должен создать hasher", func(t *testing.T) {
			hasher := NewBcryptHasher()

			require.NotNil(t, hasher)
			assert.Positive(t, hasher.cost)
		},
	)

	t.Run(
		"Должен хешировать и проверять пароль", func(t *testing.T) {
			hasher := NewBcryptHasher()

			hash, err := hasher.Hash("password-1")
			require.NoError(t, err)
			assert.NotEqual(t, "password-1", hash)

			err = hasher.Compare(hash, "password-1")
			require.NoError(t, err)
		},
	)

	t.Run(
		"Должен вернуть ошибку при неверном пароле", func(t *testing.T) {
			hasher := NewBcryptHasher()
			hash, err := hasher.Hash("password-1")
			require.NoError(t, err)

			err = hasher.Compare(hash, "wrong-password")

			require.Error(t, err)
		},
	)
}
