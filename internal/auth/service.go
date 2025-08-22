package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"

	"github.com/jourloy/nutri-backend/internal/lib"
	"github.com/jourloy/nutri-backend/internal/user"
)

const (
	argonMemory  = 64 * 1024
	argonTime    = 3
	argonThreads = 2
	saltLen      = 16
	keyLen       = 32
)

type LoginResponse struct {
	User         *user.User `json:"user"`
	AccessToken  string     `json:"-"`
	RefreshToken string     `json:"-"`
}

type Service interface {
	hashPasswordArgon2id(password string) (string, error)
	verifyPasswordArgon2id(stored, password string) (bool, error)
	Register(body RegisterData) (*LoginResponse, error)
	Login(body LoginData) (*LoginResponse, error)
	Refresh(refreshToken string) (*LoginResponse, error)
	IncreaseViewUpdates(ctx context.Context, uid string) (*user.User, error)
	Delete(id string) error
}

type service struct {
	repo        Repository
	userService user.Service
	jwtCfg      Config
}

func NewService(repo Repository) Service {
	return &service{repo: repo, userService: user.NewService(), jwtCfg: Config{
		Secret:     []byte(lib.Config.JWTSecret),
		Issuer:     "nutri-api",
		Audience:   "nutri-web",
		AccessTTL:  30 * time.Hour,
		RefreshTTL: 30 * 24 * time.Hour,
	}}
}

func (s *service) makeToken(sub string, ttl time.Duration, tokenVersion int64, tokenType string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserId:       sub,
		TokenVersion: tokenVersion,
		TokenType:    tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.jwtCfg.Issuer,
			Audience:  []string{s.jwtCfg.Audience},
			Subject:   sub,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}

	tkn := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tkn.SignedString(s.jwtCfg.Secret)
}

func (s *service) issueTokens(userID string, tokenVersion int64) (access, refresh string, err error) {
	access, err = s.makeToken(userID, s.jwtCfg.AccessTTL, tokenVersion, "access")
	if err != nil {
		return "", "", err
	}
	refresh, err = s.makeToken(userID, s.jwtCfg.RefreshTTL, tokenVersion, "refresh")
	if err != nil {
		return "", "", err
	}
	return
}

func (s *service) hashPasswordArgon2id(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, uint8(argonThreads), keyLen)

	b64 := base64.RawStdEncoding
	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonTime, argonThreads,
		b64.EncodeToString(salt),
		b64.EncodeToString(hash),
	)
	return encoded, nil
}

func (s *service) verifyPasswordArgon2id(stored, password string) (bool, error) {
	parts := strings.Split(stored, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("invalid hash format")
	}
	var mem uint32
	var time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &time, &threads); err != nil {
		return false, err
	}

	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	wantHash, err := b64.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	gotHash := argon2.IDKey([]byte(password), salt, time, mem, threads, uint32(len(wantHash)))

	// сравнение в константное время
	if subtle.ConstantTimeCompare(gotHash, wantHash) == 1 {
		return true, nil
	}
	return false, nil
}

func (s *service) Register(body RegisterData) (*LoginResponse, error) {
	hash, err := s.hashPasswordArgon2id(body.Password)
	if err != nil {
		return nil, err
	}

	u, err := s.userService.CreateUser(&user.UserCreate{
		Username:     body.Username,
		PasswordHash: hash,
	})
	if err != nil {
		return nil, err
	}

	access, refresh, err := s.issueTokens(u.Id, 1)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{User: u, AccessToken: access, RefreshToken: refresh}, nil
}

func (s *service) Login(body LoginData) (*LoginResponse, error) {
	u, err := s.userService.GetUserByUsername(body.Username)
	if err != nil {
		return nil, err
	}

	ok, err := s.verifyPasswordArgon2id(u.PasswordHash, body.Password)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("unauthorized")
	}

	access, refresh, err := s.issueTokens(u.Id, u.TokenVersion)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{User: u, AccessToken: access, RefreshToken: refresh}, nil
}

func (s *service) Refresh(refreshToken string) (*LoginResponse, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(
		refreshToken,
		&claims,
		func(t *jwt.Token) (any, error) { return s.jwtCfg.Secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(s.jwtCfg.Issuer),
		jwt.WithAudience(s.jwtCfg.Audience),
		jwt.WithLeeway(30*time.Second),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh: %w", err)
	}
	if claims.TokenType != "refresh" {
		return nil, errors.New("wrong token type")
	}

	u, err := s.userService.GetUser(claims.UserId)
	if err != nil || u == nil || u.DeletedAt != nil {
		return nil, errors.New("user not allowed")
	}

	if claims.TokenVersion != u.TokenVersion {
		return nil, errors.New("user not allowed")
	}

	access, newRefresh, err := s.issueTokens(u.Id, u.TokenVersion)
	if err != nil {
		return nil, err
	}

	err = s.userService.UpdateLogin(context.Background(), u.Id)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{User: u, AccessToken: access, RefreshToken: newRefresh}, nil
}

func (s *service) IncreaseViewUpdates(ctx context.Context, uid string) (*user.User, error) {
	return s.userService.IncreaseViewUpdates(context.Background(), uid)
}

func (s *service) Delete(id string) error {
	_, err := s.userService.DeleteUser(context.Background(), id)
	return err
}
