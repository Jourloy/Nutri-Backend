package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
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
	Secret       []byte // для HS256
	PublicKeyPEM []byte // для RS256
	Issuer       string
	Audience     string
	Leeway       time.Duration
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
}

type contextKey int

const (
	ctxAuthInfoKey contextKey = iota + 1
)

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
		// Эти опции заставят парсер валидировать iss/aud/exp/iat/nbf:
		jwt.WithIssuer(cfg.Issuer),     // если пусто — проверка пропущена
		jwt.WithAudience(cfg.Audience), // если пусто — проверка пропущена
		jwt.WithExpirationRequired(),   // требовать exp
		// по желанию: jwt.WithIssuedAt(), jwt.WithSubject("..."), etc.
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims.UserId == "" {
		return nil, errors.New("missing user id in token")
	}
	return &claims, nil
}

func WithAuthInfo(ctx context.Context, ai *AuthInfo) context.Context {
	return context.WithValue(ctx, ctxAuthInfoKey, ai)
}

func FromContext(ctx context.Context) (*AuthInfo, bool) {
	ai, ok := ctx.Value(ctxAuthInfoKey).(*AuthInfo)
	return ai, ok
}
