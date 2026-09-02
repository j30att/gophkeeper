// Package middleware содержит HTTP middleware приложения.
package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/igor/gophkeeper/internal/modules/auth/handlers/httputil"
	"github.com/igor/gophkeeper/internal/modules/auth/token"
)

type contextKey string

const (
	userIDContextKey contextKey = "user_id"
	loginContextKey  contextKey = "login"
)

// TokenValidator проверяет access token.
type TokenValidator interface {
	Validate(tokenString string) (token.Claims, error)
}

// Auth создает middleware для проверки Bearer token.
func Auth(validator TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if validator == nil {
					httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "internal server error")
					return
				}

				tokenString, err := bearerToken(r.Header.Get("Authorization"))
				if err != nil {
					httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "authorization required")
					return
				}

				claims, err := validator.Validate(tokenString)
				if err != nil {
					httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "authorization required")
					return
				}

				ctx := context.WithValue(r.Context(), userIDContextKey, claims.UserID)
				ctx = context.WithValue(ctx, loginContextKey, claims.Login)
				next.ServeHTTP(w, r.WithContext(ctx))
			},
		)
	}
}

// UserIDFromContext возвращает userID из request context.
func UserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDContextKey).(uuid.UUID)
	return userID, ok
}

// LoginFromContext возвращает login из request context.
func LoginFromContext(ctx context.Context) (string, bool) {
	login, ok := ctx.Value(loginContextKey).(string)
	return login, ok
}

func bearerToken(header string) (string, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", errors.New("missing bearer token")
	}
	tokenString := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if tokenString == "" {
		return "", errors.New("empty bearer token")
	}
	return tokenString, nil
}
