package middlewares

import (
	"net/http"

	"github.com/jourloy/nutri02/internal/auth"
)

// AdminOnly проверяет, что пользователь является администратором
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем пользователя из контекста (должен быть установлен в Auth middleware)
		user, ok := auth.UserFromContext(r.Context())
		if !ok {
			http.Error(w, `{"error": "unauthorized"}`, http.StatusUnauthorized)
			return
		}

		// Проверяем, является ли пользователь админом
		if !user.IsAdmin {
			http.Error(w, `{"error": "admin access required"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
