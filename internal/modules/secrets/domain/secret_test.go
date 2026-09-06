package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsStructuredType(t *testing.T) {
	t.Run("Должен вернуть true для JSON-типов", func(t *testing.T) {
		assert.True(t, IsStructuredType(SecretTypeCredentials))
		assert.True(t, IsStructuredType(SecretTypeCard))
	})

	t.Run("Должен вернуть false для неизвестного типа", func(t *testing.T) {
		assert.False(t, IsStructuredType(SecretType("binary")))
	})
}

func TestIsBlobType(t *testing.T) {
	t.Run("Должен вернуть true для blob-типов", func(t *testing.T) {
		assert.True(t, IsBlobType(SecretTypeText))
		assert.True(t, IsBlobType(SecretTypeBinary))
	})

	t.Run("Должен вернуть false для structured-типа", func(t *testing.T) {
		assert.False(t, IsBlobType(SecretTypeCredentials))
	})
}
