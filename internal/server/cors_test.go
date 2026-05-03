package server

import "testing"

func TestIsAllowedOrigin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		allowedOrigins []string
		origin         string
		allowed        bool
	}{
		{
			name:           "configured public origin",
			allowedOrigins: []string{"https://nutri02.com"},
			origin:         "https://nutri02.com",
			allowed:        true,
		},
		{
			name:           "loopback localhost no port",
			allowedOrigins: []string{"https://nutri02.com"},
			origin:         "http://localhost",
			allowed:        true,
		},
		{
			name:           "loopback localhost with port",
			allowedOrigins: []string{"https://nutri02.com"},
			origin:         "http://localhost:3000",
			allowed:        true,
		},
		{
			name:           "loopback ipv4",
			allowedOrigins: []string{"https://nutri02.com"},
			origin:         "http://127.0.0.1:80",
			allowed:        true,
		},
		{
			name:           "loopback ipv6",
			allowedOrigins: []string{"https://nutri02.com"},
			origin:         "https://[::1]:3001",
			allowed:        true,
		},
		{
			name:           "unrelated origin",
			allowedOrigins: []string{"https://nutri02.com"},
			origin:         "https://evil.example",
			allowed:        false,
		},
		{
			name:           "invalid origin",
			allowedOrigins: []string{"https://nutri02.com"},
			origin:         "not-a-url",
			allowed:        false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			allowed := isAllowedOrigin(tt.allowedOrigins, tt.origin)
			if allowed != tt.allowed {
				t.Fatalf("isAllowedOrigin(%q, %q) = %v, want %v", tt.allowedOrigins, tt.origin, allowed, tt.allowed)
			}
		})
	}
}
