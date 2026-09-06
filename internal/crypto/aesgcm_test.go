package crypto

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAESGCMEncryptor(t *testing.T) {
	t.Run(
		"Должен зашифровать и расшифровать данные", func(t *testing.T) {
			encryptor, err := NewAESGCMEncryptor("secret")
			require.NoError(t, err)

			ciphertext, nonce, err := encryptor.Encrypt([]byte("payload"))
			require.NoError(t, err)
			assert.NotEqual(t, []byte("payload"), ciphertext)
			assert.NotEmpty(t, nonce)

			plaintext, err := encryptor.Decrypt(ciphertext, nonce)

			require.NoError(t, err)
			assert.Equal(t, []byte("payload"), plaintext)
		},
	)

	t.Run(
		"Должен вернуть ошибку при неверном nonce", func(t *testing.T) {
			encryptor, err := NewAESGCMEncryptor("secret")
			require.NoError(t, err)
			ciphertext, _, err := encryptor.Encrypt([]byte("payload"))
			require.NoError(t, err)

			_, err = encryptor.Decrypt(ciphertext, []byte("bad"))

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidCiphertext)
		},
	)
}

func TestAESGCMEncryptorStream(t *testing.T) {
	t.Run("Должен зашифровать и расшифровать stream", func(t *testing.T) {
		encryptor, err := NewAESGCMEncryptor("secret")
		require.NoError(t, err)
		var encrypted bytes.Buffer

		size, checksum, err := encryptor.EncryptStream(strings.NewReader("payload"), &encrypted)
		require.NoError(t, err)

		assert.Equal(t, int64(7), size)
		assert.Equal(t, "239f59ed55e737c77147cf55ad0c1b030b6d7ee748a7426952f9b852d5a935e5", checksum)
		assert.NotContains(t, encrypted.String(), "payload")

		var decrypted bytes.Buffer
		err = encryptor.DecryptStream(bytes.NewReader(encrypted.Bytes()), &decrypted)

		require.NoError(t, err)
		assert.Equal(t, "payload", decrypted.String())
	})

	t.Run("Должен вернуть ошибку при неверном stream magic", func(t *testing.T) {
		encryptor, err := NewAESGCMEncryptor("secret")
		require.NoError(t, err)

		err = encryptor.DecryptStream(strings.NewReader("bad"), &bytes.Buffer{})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCiphertext)
	})

	t.Run("Должен вернуть ошибку чтения stream", func(t *testing.T) {
		encryptor, err := NewAESGCMEncryptor("secret")
		require.NoError(t, err)

		_, _, err = encryptor.EncryptStream(errorReader{}, &bytes.Buffer{})

		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})
}

type errorReader struct{}

func (errorReader) Read(_ []byte) (int, error) {
	return 0, assert.AnError
}

var _ = errors.Is
