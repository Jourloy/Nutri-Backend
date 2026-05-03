package server

import (
	"net/http"
	"strings"
)

var (
	legacyFrontHosts = map[string]string{
		"nutri.jourloy.com":   "https://nutri02.com",
		"nutri02.jourloy.com": "https://nutri02.com",
		"somivyn.com":         "https://nutri02.com",
		"www.somivyn.com":     "https://nutri02.com",
		"somivyn.jourloy.com": "https://nutri02.com",
	}
	legacyAPIHosts = map[string]string{
		"api.nutri.jourloy.com":   "https://api.nutri02.com",
		"api-nutri.jourloy.com":   "https://api.nutri02.com",
		"api.somivyn.com":         "https://api.nutri02.com",
		"api.somivyn.jourloy.com": "https://api.nutri02.com",
		"api-somivyn.jourloy.com": "https://api.nutri02.com",
	}
)

func redirectLegacyHosts(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := strings.ToLower(strings.TrimSpace(strings.Split(r.Host, ":")[0]))
		if target, ok := legacyFrontHosts[host]; ok {
			http.Redirect(w, r, target+r.URL.RequestURI(), http.StatusPermanentRedirect)
			return
		}
		if target, ok := legacyAPIHosts[host]; ok {
			http.Redirect(w, r, target+r.URL.RequestURI(), http.StatusPermanentRedirect)
			return
		}

		next.ServeHTTP(w, r)
	})
}
