package middlewares

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/jourloy/nutri02/internal/auth"
	"github.com/jourloy/nutri02/internal/lib"
	"github.com/jourloy/nutri02/internal/user"
)

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Public auth endpoints must remain reachable even with stale/invalid jwt cookie.
		// Otherwise login/refresh flows are blocked by the middleware itself.
		if isPublicAuthRoute(r.Method, r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		jwtCfg := auth.Config{
			Secret:            []byte(lib.Config.JWTSecret),
			Issuer:            "nutri02-api",
			Audience:          "nutri02-web",
			AcceptedIssuers:   []string{"somivyn-api"},
			AcceptedAudiences: []string{"somivyn-web"},
			Leeway:            30 * time.Second,
			AccessTTL:         1 * time.Hour,
			RefreshTTL:        30 * 24 * time.Hour,
		}

		token := extractToken(r)
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}

		userRepo := user.NewRepository()
		claims, err := auth.ValidateToken(jwtCfg, token)
		if err != nil {
			// Если access токен просто истёк — чистим ТОЛЬКО access cookie (оставляем refresh, чтобы фронт мог рефрешнуться)
			if errors.Is(err, jwt.ErrTokenExpired) {
				auth.ClearAccessCookie(w, r)
			} else {
				// Невалиден (подпись/формат и т.п.) — чистим всё
				auth.ClearAuthCookies(w, r)
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
			auth.ClearAuthCookies(w, r)
			http.Error(w, "failed to load user", http.StatusForbidden)
			return
		}
		if u == nil || u.DeletedAt != nil {
			auth.ClearAuthCookies(w, r)
			http.Error(w, "user disabled", http.StatusForbidden)
			return
		}
		if u.TokenVersion != ai.TokenVersion {
			// Версия токена не совпала (например, сменили пароль/выход со всех устройств) — чистим всё
			auth.ClearAuthCookies(w, r)
			http.Error(w, "token version incorrect", http.StatusForbidden)
			return
		}

		ctx = auth.ContextWithUser(ctx, *u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublicAuthRoute(method, path string) bool {
	if method == "" {
		return false
	}

	normalized := strings.TrimSuffix(path, "/")
	switch {
	case method == http.MethodPost && normalized == "/api/v1/auth/login":
		return true
	case method == http.MethodPost && normalized == "/api/v1/auth/register":
		return true
	case method == http.MethodPost && normalized == "/api/v1/auth/refresh":
		return true
	case method == http.MethodPost && normalized == "/api/v1/auth/request-password-reset":
		return true
	case method == http.MethodPost && normalized == "/api/v1/auth/reset-password":
		return true
	case method == http.MethodGet && strings.HasPrefix(normalized, "/api/v1/auth/check-username/"):
		return true
	case method == http.MethodGet && normalized == "/api/v1/auth/validate-reset-token":
		return true
	default:
		return false
	}
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
