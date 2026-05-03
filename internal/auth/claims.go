package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AuthInfo struct {
	UserId       string
	TokenVersion int64
	Claims       *Claims
	Token        string
}

type Claims struct {
	UserId       string `json:"id"`
	TokenVersion int64  `json:"tokenVersion"`
	TokenType    string `json:"tokenType"` // "access" или "refresh"
	jwt.RegisteredClaims
}

type Config struct {
	Secret            []byte // для HS256
	PublicKeyPEM      []byte // для RS256
	Issuer            string
	Audience          string
	AcceptedIssuers   []string
	AcceptedAudiences []string
	Leeway            time.Duration
	AccessTTL         time.Duration
	RefreshTTL        time.Duration
}

type contextKey int

const (
	ctxAuthInfoKey contextKey = iota + 1
)

// AuthContextKey для доступа к Claims из контекста
const AuthContextKey = ctxAuthInfoKey

func parseRSAPublicKeyFromPEM(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid PEM")
	}
	pubAny, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := pubAny.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not RSA public key")
	}
	return pub, nil
}

func ValidateToken(cfg Config, tokenString string) (*Claims, error) {
	var keyFunc jwt.Keyfunc
	var validAlgs []string

	switch {
	case len(cfg.Secret) > 0:
		validAlgs = []string{jwt.SigningMethodHS256.Alg()}
		keyFunc = func(t *jwt.Token) (any, error) { return cfg.Secret, nil }
	case len(cfg.PublicKeyPEM) > 0:
		validAlgs = []string{jwt.SigningMethodRS256.Alg()}
		pub, err := parseRSAPublicKeyFromPEM(cfg.PublicKeyPEM)
		if err != nil {
			return nil, err
		}
		keyFunc = func(t *jwt.Token) (any, error) { return pub, nil }
	default:
		return nil, errors.New("no JWT key configured")
	}

	var claims Claims
	_, err := jwt.ParseWithClaims(
		tokenString,
		&claims,
		keyFunc,
		// ВКЛЮЧАЕМ ПРОВЕРКИ КЛЕЙМОВ ЗДЕСЬ:
		jwt.WithValidMethods(validAlgs),
		jwt.WithLeeway(cfg.Leeway),
		jwt.WithExpirationRequired(), // требовать exp
		// по желанию: jwt.WithIssuedAt(), jwt.WithSubject("..."), etc.
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if err := validateRegisteredClaims(cfg, &claims); err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims.UserId == "" {
		return nil, errors.New("missing user id in token")
	}
	return &claims, nil
}

func validateRegisteredClaims(cfg Config, claims *Claims) error {
	if claims == nil {
		return errors.New("missing claims")
	}

	acceptedIssuers := appendAcceptedValues(cfg.AcceptedIssuers, cfg.Issuer)
	if len(acceptedIssuers) > 0 && !containsFold(acceptedIssuers, claims.Issuer) {
		return fmt.Errorf("invalid issuer %q", claims.Issuer)
	}

	acceptedAudiences := appendAcceptedValues(cfg.AcceptedAudiences, cfg.Audience)
	if len(acceptedAudiences) > 0 {
		audiences := []string(claims.Audience)
		if !intersectsFold(acceptedAudiences, audiences) {
			return fmt.Errorf("invalid audience %q", audiences)
		}
	}

	return nil
}

func appendAcceptedValues(values []string, canonical string) []string {
	result := make([]string, 0, len(values)+1)
	seen := make(map[string]struct{}, len(values)+1)

	appendValue := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}

		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			return
		}

		seen[key] = struct{}{}
		result = append(result, trimmed)
	}

	appendValue(canonical)
	for _, value := range values {
		appendValue(value)
	}

	return result
}

func containsFold(values []string, target string) bool {
	normalizedTarget := strings.TrimSpace(target)
	if normalizedTarget == "" {
		return false
	}

	for _, value := range values {
		if strings.EqualFold(value, normalizedTarget) {
			return true
		}
	}

	return false
}

func intersectsFold(values []string, targets []string) bool {
	for _, target := range targets {
		if containsFold(values, target) {
			return true
		}
	}

	return false
}

func WithAuthInfo(ctx context.Context, ai *AuthInfo) context.Context {
	return context.WithValue(ctx, ctxAuthInfoKey, ai)
}

func FromContext(ctx context.Context) (*AuthInfo, bool) {
	ai, ok := ctx.Value(ctxAuthInfoKey).(*AuthInfo)
	return ai, ok
}
