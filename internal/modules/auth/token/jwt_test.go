package token

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTIssuer(t *testing.T) {
	t.Run(
		"Должен создать issuer", func(t *testing.T) {
			issuer := NewJWTIssuer("secret", time.Hour)

			require.NotNil(t, issuer)
			assert.Equal(t, []byte("secret"), issuer.secret)
			assert.Equal(t, time.Hour, issuer.ttl)
		},
	)

	t.Run(
		"Должен выпустить валидный JWT", func(t *testing.T) {
			userID := uuid.New()
			issuer := NewJWTIssuer("secret", time.Hour)

			tokenString, err := issuer.Issue(userID, "igor")
			require.NoError(t, err)
			assert.NotEmpty(t, tokenString)

			parsed, err := jwt.Parse(
				tokenString,
				func(_ *jwt.Token) (any, error) { return []byte("secret"), nil },
				jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			)

			require.NoError(t, err)
			require.True(t, parsed.Valid)
			claims, ok := parsed.Claims.(jwt.MapClaims)
			require.True(t, ok)
			assert.Equal(t, userID.String(), claims["sub"])
			assert.Equal(t, "igor", claims["login"])
			assert.NotEmpty(t, claims["iat"])
			assert.NotEmpty(t, claims["exp"])
		},
	)

	t.Run(
		"Должен проверить валидный JWT", func(t *testing.T) {
			userID := uuid.New()
			issuer := NewJWTIssuer("secret", time.Hour)
			tokenString, err := issuer.Issue(userID, "igor")
			require.NoError(t, err)

			claims, err := issuer.Validate(tokenString)

			require.NoError(t, err)
			assert.Equal(t, userID, claims.UserID)
			assert.Equal(t, "igor", claims.Login)
		},
	)

	t.Run(
		"Должен вернуть ошибку при невалидном JWT", func(t *testing.T) {
			issuer := NewJWTIssuer("secret", time.Hour)

			_, err := issuer.Validate("bad-token")

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidToken)
		},
	)

	t.Run(
		"Должен вернуть ошибку при неверной подписи JWT", func(t *testing.T) {
			userID := uuid.New()
			issuer := NewJWTIssuer("secret", time.Hour)
			anotherIssuer := NewJWTIssuer("another-secret", time.Hour)
			tokenString, err := anotherIssuer.Issue(userID, "igor")
			require.NoError(t, err)

			_, err = issuer.Validate(tokenString)

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidToken)
		},
	)

	t.Run(
		"Должен вернуть ошибку при истекшем JWT", func(t *testing.T) {
			userID := uuid.New()
			issuer := NewJWTIssuer("secret", -time.Hour)
			tokenString, err := issuer.Issue(userID, "igor")
			require.NoError(t, err)

			_, err = issuer.Validate(tokenString)

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidToken)
		},
	)

	t.Run(
		"Должен вернуть ошибку без login claim", func(t *testing.T) {
			issuer := NewJWTIssuer("secret", time.Hour)
			tokenString, err := jwt.NewWithClaims(
				jwt.SigningMethodHS256,
				jwt.MapClaims{
					"sub": uuid.New().String(),
					"exp": time.Now().Add(time.Hour).Unix(),
				},
			).SignedString([]byte("secret"))
			require.NoError(t, err)

			_, err = issuer.Validate(tokenString)

			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInvalidToken)
		},
	)
}
