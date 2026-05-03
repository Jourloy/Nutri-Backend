package auth

import (
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	AccessCookieName  = "jwt"
	AccessCookiePath  = "/"
	RefreshCookieName = "refresh_token"
	RefreshCookiePath = "/api/v1/auth/refresh"
)

type CookiePolicy struct {
	Secure   bool
	SameSite http.SameSite
}

func ResolveCookiePolicy(r *http.Request) CookiePolicy {
	if requestScheme(r) != "https" && IsLoopbackHost(requestHost(r)) {
		return CookiePolicy{
			Secure:   false,
			SameSite: http.SameSiteLaxMode,
		}
	}

	return CookiePolicy{
		Secure:   true,
		SameSite: http.SameSiteNoneMode,
	}
}

func SetAuthCookies(w http.ResponseWriter, r *http.Request, access, refresh string) {
	policy := ResolveCookiePolicy(r)
	setCookie(w, policy, AccessCookieName, access, AccessCookiePath, false)
	setCookie(w, policy, RefreshCookieName, refresh, RefreshCookiePath, false)
}

func ClearAuthCookies(w http.ResponseWriter, r *http.Request) {
	ClearAccessCookie(w, r)
	ClearRefreshCookie(w, r)
}

func ClearAccessCookie(w http.ResponseWriter, r *http.Request) {
	policy := ResolveCookiePolicy(r)
	setCookie(w, policy, AccessCookieName, "", AccessCookiePath, true)
}

func ClearRefreshCookie(w http.ResponseWriter, r *http.Request) {
	policy := ResolveCookiePolicy(r)
	setCookie(w, policy, RefreshCookieName, "", RefreshCookiePath, true)
}

func IsLoopbackHost(host string) bool {
	normalized := strings.TrimSpace(strings.ToLower(host))
	if normalized == "" {
		return false
	}

	if normalized == "localhost" {
		return true
	}

	ip := net.ParseIP(normalized)
	return ip != nil && ip.IsLoopback()
}

func setCookie(
	w http.ResponseWriter,
	policy CookiePolicy,
	name string,
	value string,
	path string,
	expire bool,
) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		HttpOnly: true,
		SameSite: policy.SameSite,
		Secure:   policy.Secure,
	}

	if expire {
		cookie.MaxAge = -1
		cookie.Expires = time.Unix(0, 0)
	}

	http.SetCookie(w, cookie)
}

func requestScheme(r *http.Request) string {
	forwardedProto := firstHeaderValue(r.Header.Get("X-Forwarded-Proto"))
	if forwardedProto != "" {
		return strings.ToLower(forwardedProto)
	}

	if r.TLS != nil {
		return "https"
	}

	return "http"
}

func requestHost(r *http.Request) string {
	host := firstHeaderValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}

	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}

	if normalizedHost, _, err := net.SplitHostPort(host); err == nil {
		return strings.Trim(normalizedHost, "[]")
	}

	return strings.Trim(host, "[]")
}

func firstHeaderValue(value string) string {
	if value == "" {
		return ""
	}

	return strings.TrimSpace(strings.Split(value, ",")[0])
}
