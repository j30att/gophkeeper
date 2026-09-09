// Package password предоставляет сервисы хеширования паролей.
package password

import "golang.org/x/crypto/bcrypt"

// BcryptHasher хеширует пароли через bcrypt.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher создает BcryptHasher со стандартной стоимостью bcrypt.
func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{cost: bcrypt.DefaultCost}
}

// Hash возвращает bcrypt-хеш пароля.
func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Compare проверяет, что пароль соответствует хешу.
func (_ *BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}
