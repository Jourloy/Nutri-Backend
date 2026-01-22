package middlewares

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/jourloy/nutri-backend/internal/lib"
)

type serviceAuthKey struct{}

// ServiceAuthInfo contains information about service authentication
type ServiceAuthInfo struct {
	Authenticated bool
}

// ServiceAuth middleware validates service token for internal endpoints
func ServiceAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractServiceToken(r)
		if token == "" {
			http.Error(w, "service token required", http.StatusUnauthorized)
			return
		}

		if lib.Config.ServiceToken == "" {
			http.Error(w, "service auth not configured", http.StatusServiceUnavailable)
			return
		}

		// Constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(token), []byte(lib.Config.ServiceToken)) != 1 {
			http.Error(w, "invalid service token", http.StatusUnauthorized)
			return
		}

		// Set service auth info in context
		ctx := context.WithValue(r.Context(), serviceAuthKey{}, &ServiceAuthInfo{Authenticated: true})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractServiceToken extracts the service token from the request
func extractServiceToken(r *http.Request) string {
	// Check X-Service-Token header first
	if token := r.Header.Get("X-Service-Token"); token != "" {
		return strings.TrimSpace(token)
	}

	// Fallback to Authorization header with Bearer prefix
	if ah := r.Header.Get("Authorization"); ah != "" {
		parts := strings.Fields(ah)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Service") {
			return parts[1]
		}
	}

	return ""
}

// ServiceAuthFromContext retrieves service auth info from context
func ServiceAuthFromContext(ctx context.Context) (*ServiceAuthInfo, bool) {
	info, ok := ctx.Value(serviceAuthKey{}).(*ServiceAuthInfo)
	return info, ok
}
