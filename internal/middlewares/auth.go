package middlewares

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/jourloy/nutri-backend/internal/auth"
	"github.com/jourloy/nutri-backend/internal/lib"
	"github.com/jourloy/nutri-backend/internal/user"
)

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userRepo := user.NewRepository()
		jwtCfg := auth.Config{
			Secret:     []byte(lib.Config.JWTSecret),
			Issuer:     "nutri-api",
			Audience:   "nutri-web",
			AccessTTL:  1 * time.Hour,
			RefreshTTL: 30 * 24 * time.Hour,
		}

		token := extractToken(r)
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}

		claims, err := auth.ValidateToken(jwtCfg, token)
		if err != nil {
			// Если access токен просто истёк — чистим ТОЛЬКО access cookie (оставляем refresh, чтобы фронт мог рефрешнуться)
			if errors.Is(err, jwt.ErrTokenExpired) {
				clearAccessCookie(w, r)
			} else {
				// Невалиден (подпись/формат и т.п.) — чистим всё
				clearAllAuthCookies(w, r)
			}
			w.Header().Set("WWW-Authenticate", "Bearer error=\"invalid_token\"")
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
			return
		}

		ai := &auth.AuthInfo{
			UserId:       claims.UserId,
			TokenVersion: claims.TokenVersion,
			Claims:       claims,
			Token:        token,
		}
		ctx := auth.WithAuthInfo(r.Context(), ai)

		u, err := userRepo.GetUser(ctx, claims.UserId)
		if err != nil {
			clearAllAuthCookies(w, r)
			http.Error(w, "failed to load user", http.StatusForbidden)
			return
		}
		if u == nil || u.DeletedAt != nil {
			clearAllAuthCookies(w, r)
			http.Error(w, "user disabled", http.StatusForbidden)
			return
		}
		if u.TokenVersion != ai.TokenVersion {
			// Версия токена не совпала (например, сменили пароль/выход со всех устройств) — чистим всё
			clearAllAuthCookies(w, r)
			http.Error(w, "token version incorrect", http.StatusForbidden)
			return
		}

		ctx = auth.ContextWithUser(ctx, *u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractToken(r *http.Request) string {
	if ah := r.Header.Get("Authorization"); ah != "" {
		parts := strings.Fields(ah)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return parts[1]
		}
	}
	if c, err := r.Cookie("jwt"); err == nil {
		return strings.TrimSpace(c.Value)
	}
	return ""
}

// ===== Очистка cookie =====

// Подбери SameSite/Secure под свои условия. Если у тебя cross-site (frontend другой домен)
// и ты ставишь SameSite=None, здесь тоже нужно SameSite=None и Secure=true.
func clearAccessCookie(w http.ResponseWriter, r *http.Request) {
	secure, samesite := cookieFlags(r) // см. функцию ниже
	http.SetCookie(w, &http.Cookie{
		Name:     "jwt",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: samesite,
		Secure:   secure,
		// Любой из двух вариантов — MaxAge=-1 или Expires в прошлом — удаляет cookie
		MaxAge:  -1,
		Expires: time.Unix(0, 0),
	})
}

func clearRefreshCookie(w http.ResponseWriter, r *http.Request) {
	secure, samesite := cookieFlags(r)
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/auth/refresh", // тот же Path, что при установке!
		HttpOnly: true,
		SameSite: samesite,
		Secure:   secure,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func clearAllAuthCookies(w http.ResponseWriter, r *http.Request) {
	clearAccessCookie(w, r)
	clearRefreshCookie(w, r)
}

// Определи флаги cookie. Для локалки (same-site на http://localhost) — Lax,false.
// Для cross-site (другие домены) — None,true (иначе браузер проигнорирует).
func cookieFlags(r *http.Request) (secure bool, samesite http.SameSite) {
	return true, http.SameSiteNoneMode
}
