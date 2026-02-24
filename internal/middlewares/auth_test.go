package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsPublicAuthRoute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		want   bool
	}{
		{name: "login", method: http.MethodPost, path: "/api/v1/auth/login", want: true},
		{name: "login with trailing slash", method: http.MethodPost, path: "/api/v1/auth/login/", want: true},
		{name: "register", method: http.MethodPost, path: "/api/v1/auth/register", want: true},
		{name: "refresh", method: http.MethodPost, path: "/api/v1/auth/refresh", want: true},
		{name: "request password reset", method: http.MethodPost, path: "/api/v1/auth/request-password-reset", want: true},
		{name: "reset password", method: http.MethodPost, path: "/api/v1/auth/reset-password", want: true},
		{name: "validate reset token", method: http.MethodGet, path: "/api/v1/auth/validate-reset-token", want: true},
		{name: "check username", method: http.MethodGet, path: "/api/v1/auth/check-username/nutri", want: true},
		{name: "check username with trailing slash", method: http.MethodGet, path: "/api/v1/auth/check-username/nutri/", want: true},
		{name: "auth me is protected", method: http.MethodPost, path: "/api/v1/auth/me", want: false},
		{name: "same path wrong method", method: http.MethodGet, path: "/api/v1/auth/login", want: false},
		{name: "non auth path", method: http.MethodGet, path: "/api/v1/product/today", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := isPublicAuthRoute(tt.method, tt.path)
			if got != tt.want {
				t.Fatalf("isPublicAuthRoute(%q, %q) = %v, want %v", tt.method, tt.path, got, tt.want)
			}
		})
	}
}

func TestAuthMiddleware_AllowsPublicLoginRouteWithInvalidToken(t *testing.T) {
	t.Parallel()

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := Auth(next)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: "not-a-jwt"})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !nextCalled {
		t.Fatal("expected next handler to be called for public auth route")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}

func TestAuthMiddleware_BlocksProtectedRouteWithInvalidToken(t *testing.T) {
	t.Parallel()

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})

	handler := Auth(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/product/today", nil)
	req.AddCookie(&http.Cookie{Name: "jwt", Value: "not-a-jwt"})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if nextCalled {
		t.Fatal("did not expect next handler to be called for protected route with invalid token")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}
