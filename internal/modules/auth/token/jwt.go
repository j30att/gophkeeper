// Package token предоставляет сервисы access token.
package token

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrInvalidToken означает, что access token некорректен.
var ErrInvalidToken = errors.New("invalid token")

// Claims содержит проверенные данные access token.
type Claims struct {
	UserID uuid.UUID
	Login  string
}

// JWTIssuer выпускает HMAC-подписанные JWT access tokens.
type JWTIssuer struct {
	secret []byte
	ttl    time.Duration
}

// NewJWTIssuer создает JWTIssuer.
func NewJWTIssuer(secret string, ttl time.Duration) *JWTIssuer {
	return &JWTIssuer{secret: []byte(secret), ttl: ttl}
}

// Issue создает access token для userID и login.
func (i *JWTIssuer) Issue(userID uuid.UUID, login string) (string, error) {
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub":   userID.String(),
		"login": login,
		"iat":   now.Unix(),
		"exp":   now.Add(i.ttl).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
}

// Validate проверяет access token и возвращает его claims.
func (i *JWTIssuer) Validate(tokenString string) (Claims, error) {
	parsed, err := jwt.Parse(
		tokenString,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}
			return i.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	if !parsed.Valid {
		return Claims{}, ErrInvalidToken
	}

	mapClaims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return Claims{}, ErrInvalidToken
	}

	userIDRaw, err := mapClaims.GetSubject()
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}
	userID, err := uuid.Parse(userIDRaw)
	if err != nil {
		return Claims{}, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	login, ok := mapClaims["login"].(string)
	if !ok || login == "" {
		return Claims{}, ErrInvalidToken
	}

	return Claims{UserID: userID, Login: login}, nil
}
