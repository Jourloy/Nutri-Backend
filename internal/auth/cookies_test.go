package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveCookiePolicyUsesLoopbackFlagsOverHTTP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
	}{
		{name: "localhost", url: "http://localhost:3002/api/v1/auth/login"},
		{name: "ipv4", url: "http://127.0.0.1:3002/api/v1/auth/login"},
		{name: "ipv6", url: "http://[::1]:3002/api/v1/auth/login"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, tt.url, nil)
			policy := ResolveCookiePolicy(req)

			if policy.Secure {
				t.Fatal("expected loopback HTTP cookie policy to disable Secure")
			}
			if policy.SameSite != http.SameSiteLaxMode {
				t.Fatalf("expected SameSite Lax, got %v", policy.SameSite)
			}
		})
	}
}

func TestSetAndClearAuthCookiesUseLoopbackFlags(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "http://localhost:3002/api/v1/auth/login", nil)

	rrSet := httptest.NewRecorder()
	SetAuthCookies(rrSet, req, "access-token", "refresh-token")
	assertCookieFlags(t, rrSet.Result().Cookies(), AccessCookieName, AccessCookiePath, false, http.SameSiteLaxMode, false)
	assertCookieFlags(t, rrSet.Result().Cookies(), RefreshCookieName, RefreshCookiePath, false, http.SameSiteLaxMode, false)

	rrClear := httptest.NewRecorder()
	ClearAuthCookies(rrClear, req)
	assertCookieFlags(t, rrClear.Result().Cookies(), AccessCookieName, AccessCookiePath, false, http.SameSiteLaxMode, true)
	assertCookieFlags(t, rrClear.Result().Cookies(), RefreshCookieName, RefreshCookiePath, false, http.SameSiteLaxMode, true)
}

func TestSetAuthCookiesUseProductionFlagsOnHTTPS(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "https://nutri02.com/api/v1/auth/login", nil)
	rr := httptest.NewRecorder()

	SetAuthCookies(rr, req, "access-token", "refresh-token")

	assertCookieFlags(t, rr.Result().Cookies(), AccessCookieName, AccessCookiePath, true, http.SameSiteNoneMode, false)
	assertCookieFlags(t, rr.Result().Cookies(), RefreshCookieName, RefreshCookiePath, true, http.SameSiteNoneMode, false)
}

func assertCookieFlags(
	t *testing.T,
	cookies []*http.Cookie,
	name string,
	path string,
	secure bool,
	sameSite http.SameSite,
	expectExpired bool,
) {
	t.Helper()

	for _, cookie := range cookies {
		if cookie.Name != name {
			continue
		}

		if cookie.Path != path {
			t.Fatalf("expected cookie %q path %q, got %q", name, path, cookie.Path)
		}
		if cookie.Secure != secure {
			t.Fatalf("expected cookie %q Secure=%v, got %v", name, secure, cookie.Secure)
		}
		if cookie.SameSite != sameSite {
			t.Fatalf("expected cookie %q SameSite=%v, got %v", name, sameSite, cookie.SameSite)
		}
		if expectExpired && cookie.MaxAge >= 0 {
			t.Fatalf("expected cookie %q to be expired, got MaxAge=%d", name, cookie.MaxAge)
		}
		if !expectExpired && cookie.MaxAge < 0 {
			t.Fatalf("expected cookie %q not to be expired, got MaxAge=%d", name, cookie.MaxAge)
		}
		return
	}

	t.Fatalf("expected cookie %q", name)
}
