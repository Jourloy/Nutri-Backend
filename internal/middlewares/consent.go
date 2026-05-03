package middlewares

import (
	"encoding/json"
	"net/http"

	"github.com/jourloy/nutri02/internal/auth"
	"github.com/jourloy/nutri02/internal/consent"
)

func RequireConsent(consentType string) func(http.Handler) http.Handler {
	service := consent.NewService(consent.NewRepository())

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := auth.UserFromContext(r.Context())
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if user.IsAdmin {
				next.ServeHTTP(w, r)
				return
			}

			granted, err := service.HasGrantedConsent(r.Context(), user.Id, consentType)
			if err != nil {
				http.Error(w, "failed to verify consent", http.StatusInternalServerError)
				return
			}
			if !granted {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error":       "consent_required",
					"consentType": consentType,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
