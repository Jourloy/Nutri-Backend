package middlewares

import (
	"net/http"
	"strings"
	"time"

	"github.com/jourloy/nutri-backend/internal/auth"
	"github.com/jourloy/nutri-backend/internal/lib"
	"github.com/jourloy/nutri-backend/internal/user"
)

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userRepo := user.NewRepository()
		jwt := auth.Config{
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

		claims, err := auth.ValidateToken(jwt, token)
		if err != nil {
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
			http.Error(w, "failed to load user", http.StatusForbidden)
			return
		}

		if u == nil || u.DeletedAt != nil {
			http.Error(w, "user disabled", http.StatusForbidden)
			return
		}

		if u.TokenVersion != ai.TokenVersion {
			http.Error(w, "token version incorrect", http.StatusForbidden)
			return
		}

		ctx = auth.ContextWithUser(ctx, *u)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func extractToken(r *http.Request) string {
	ah := r.Header.Get("Authorization")
	if ah != "" {
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
