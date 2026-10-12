// Package middleware содержит middleware для JWT-аутентификации
// и работы с контекстом запроса.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/knapptan/go-musthave-diploma-tpl/internal/auth"
)

type ctxKey int

const userIDKey ctxKey = 1

// UserID извлекает userID из контекста.
func UserID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey).(string)
	return v, ok
}

// Auth — JWT middleware, проверяющая заголовок Authorization: Bearer <token>.
func Auth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			uid, err := auth.ParseToken(secret, strings.TrimPrefix(h, "Bearer "))
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey, uid)))
		})
	}
}
