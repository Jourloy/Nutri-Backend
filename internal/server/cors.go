package server

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/cors"

	"github.com/jourloy/nutri02/internal/auth"
)

func newCORSOptions(allowedOrigins []string) cors.Options {
	return cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			return isAllowedOrigin(allowedOrigins, origin)
		},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}
}

func isAllowedOrigin(allowedOrigins []string, origin string) bool {
	normalizedOrigin := normalizeOrigin(origin)
	if normalizedOrigin == "" {
		return false
	}

	for _, allowedOrigin := range allowedOrigins {
		if normalizeOrigin(allowedOrigin) == normalizedOrigin {
			return true
		}
	}

	parsedOrigin, err := url.Parse(normalizedOrigin)
	if err != nil {
		return false
	}

	switch parsedOrigin.Scheme {
	case "http", "https":
		return auth.IsLoopbackHost(parsedOrigin.Hostname())
	default:
		return false
	}
}

func normalizeOrigin(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	parsedValue, err := url.Parse(trimmed)
	if err != nil || parsedValue.Scheme == "" || parsedValue.Host == "" {
		return ""
	}

	return strings.ToLower(parsedValue.Scheme) + "://" + strings.ToLower(parsedValue.Host)
}
