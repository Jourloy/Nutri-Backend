package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"

	"github.com/jourloy/nutri02/internal/email"
	"github.com/jourloy/nutri02/internal/lib"
	"github.com/jourloy/nutri02/internal/telegram"
	"github.com/jourloy/nutri02/internal/user"
	"github.com/jourloy/nutri02/pkg/validator"
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
	Logout(id string) error
	Delete(id string) error
	UpdateLocale(ctx context.Context, uid string, locale string) (*user.User, error)
	CheckUsernameAvailability(username string) (bool, error)
	// Password reset
	RequestPasswordReset(ctx context.Context, username string) (*PasswordResetResponse, error)
	ValidateResetToken(ctx context.Context, token string) (bool, error)
	ResetPassword(ctx context.Context, token string, newPassword string) error
}

type service struct {
	repo            Repository
	userService     user.Service
	telegramService telegram.Service
	emailService    email.Service
	jwtCfg          Config
}

func NewService(repo Repository) Service {
	emailSvc := email.NewService()
	return &service{
		repo:            repo,
		userService:     user.NewService(),
		telegramService: telegram.NewService(),
		emailService:    emailSvc,
		jwtCfg: Config{
			Secret:            []byte(lib.Config.JWTSecret),
			Issuer:            "nutri02-api",
			Audience:          "nutri02-web",
			AcceptedIssuers:   []string{"somivyn-api"},
			AcceptedAudiences: []string{"somivyn-web"},
			Leeway:            30 * time.Second,
			AccessTTL:         30 * time.Hour,
			RefreshTTL:        30 * 24 * time.Hour,
		},
	}
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
	// Validate username format
	if errMsg := validator.ValidateUsername(body.Username); errMsg != "" {
		return nil, errors.New(errMsg)
	}
	if !body.Is18 {
		return nil, errors.New("age confirmation is required")
	}
	if !body.AcceptTerms {
		return nil, errors.New("terms acceptance is required")
	}
	if !body.AcceptPrivacy {
		return nil, errors.New("privacy policy acceptance is required")
	}
	if !body.PersonalDataConsent {
		return nil, errors.New("personal data processing consent is required")
	}
	if strings.TrimSpace(body.PersonalDataConsentVersion) == "" {
		return nil, errors.New("personal data consent version is required")
	}

	hash, err := s.hashPasswordArgon2id(body.Password)
	if err != nil {
		return nil, err
	}

	locale := strings.ToLower(body.Locale)
	if locale != "en" && locale != "ru" {
		locale = "ru"
	}

	u, err := s.userService.CreateUser(&user.UserCreate{
		Username:        strings.ToLower(body.Username),
		PasswordHash:    hash,
		Locale:          &locale,
		IsAcceptTerms:   body.AcceptTerms,
		IsAcceptPrivacy: body.AcceptPrivacy,
		Is18:            body.Is18,
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
	u, err := s.userService.GetUserByUsername(strings.ToLower(body.Username))
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
	claims, err := ValidateToken(s.jwtCfg, refreshToken)
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

func (s *service) Logout(id string) error {
	return s.userService.InvalidateTokens(context.Background(), id)
}

func (s *service) Delete(id string) error {
	_, err := s.userService.DeleteUser(context.Background(), id)
	return err
}

func (s *service) UpdateLocale(ctx context.Context, uid string, locale string) (*user.User, error) {
	return s.userService.UpdateLocale(ctx, uid, locale)
}

func (s *service) CheckUsernameAvailability(username string) (bool, error) {
	u, err := s.userService.GetUserByUsername(strings.ToLower(username))
	if err != nil {
		return false, err
	}
	// если пользователь не найден, username доступен
	return u == nil, nil
}

// RequestPasswordReset initiates password reset process
func (s *service) RequestPasswordReset(ctx context.Context, username string) (*PasswordResetResponse, error) {
	// Find user by username
	u, err := s.userService.GetUserByUsername(strings.ToLower(username))
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errors.New("user not found")
	}

	// Check if user has Telegram linked
	tgProfile, _ := s.telegramService.GetByUserId(ctx, u.Id)
	if tgProfile != nil && tgProfile.TelegramId != nil && *tgProfile.TelegramId != "" {
		// Send via Telegram
		token, err := s.repo.CreatePasswordResetToken(ctx, u.Id, "telegram")
		if err != nil {
			return nil, err
		}

		resetURL := fmt.Sprintf("%s/reset-password?token=%s", lib.Config.FrontURL, token.Token)
		err = s.sendTelegramPasswordReset(*tgProfile.TelegramId, resetURL, u.Locale)
		if err != nil {
			return nil, fmt.Errorf("failed to send telegram message: %w", err)
		}

		return &PasswordResetResponse{
			Method:  "telegram",
			Message: "Password reset link sent to your Telegram",
			Sent:    true,
		}, nil
	}

	// Check if user has email
	if u.Email != nil && *u.Email != "" {
		// Check rate limit (max 2 per day)
		count, err := s.repo.GetPasswordResetEmailCountToday(ctx, u.Id)
		if err != nil {
			return nil, err
		}
		if count >= 2 {
			return &PasswordResetResponse{
				Method:  "email",
				Message: "Email limit reached. Please try again tomorrow or link your Telegram account.",
				Sent:    false,
			}, nil
		}

		// Send via Email
		token, err := s.repo.CreatePasswordResetToken(ctx, u.Id, "email")
		if err != nil {
			return nil, err
		}

		resetURL := fmt.Sprintf("%s/reset-password?token=%s", lib.Config.FrontURL, token.Token)
		err = s.sendEmailPasswordReset(*u.Email, resetURL, u.Locale)
		if err != nil {
			return nil, fmt.Errorf("failed to send email: %w", err)
		}

		// Increment email count
		if err := s.repo.IncrementPasswordResetEmailCount(ctx, u.Id); err != nil {
			// Log but don't fail
		}

		// Mask email for response
		maskedEmail := maskEmail(*u.Email)

		return &PasswordResetResponse{
			Method:  "email",
			Message: "Password reset link sent to your email",
			Sent:    true,
			Email:   maskedEmail,
		}, nil
	}

	return &PasswordResetResponse{
		Method:  "none",
		Message: "No recovery method available. Please contact support.",
		Sent:    false,
	}, nil
}

// ValidateResetToken checks if a token is valid
func (s *service) ValidateResetToken(ctx context.Context, token string) (bool, error) {
	prt, err := s.repo.GetPasswordResetToken(ctx, token)
	if err != nil {
		return false, nil
	}

	// Check if expired
	if time.Now().After(prt.ExpiresAt) {
		return false, nil
	}

	// Check if already used
	if prt.UsedAt != nil {
		return false, nil
	}

	return true, nil
}

// ResetPassword sets a new password using the reset token
func (s *service) ResetPassword(ctx context.Context, token string, newPassword string) error {
	// Validate token
	prt, err := s.repo.GetPasswordResetToken(ctx, token)
	if err != nil {
		return errors.New("invalid token")
	}

	if time.Now().After(prt.ExpiresAt) {
		return errors.New("token expired")
	}

	if prt.UsedAt != nil {
		return errors.New("token already used")
	}

	// Hash new password
	hash, err := s.hashPasswordArgon2id(newPassword)
	if err != nil {
		return err
	}

	// Update password in user repository
	if err := s.updateUserPassword(ctx, prt.UserId, hash); err != nil {
		return err
	}

	// Mark token as used
	if err := s.repo.MarkPasswordResetTokenUsed(ctx, prt.Id); err != nil {
		// Log but don't fail
	}

	// Invalidate all existing tokens (force re-login)
	if err := s.userService.InvalidateTokens(ctx, prt.UserId); err != nil {
		// Log but don't fail
	}

	// Clean up expired tokens asynchronously
	go s.repo.DeleteExpiredPasswordResetTokens(context.Background())

	return nil
}

// updateUserPassword updates user's password hash
func (s *service) updateUserPassword(ctx context.Context, userId string, passwordHash string) error {
	// We need to add this method to user repository
	return user.NewRepository().UpdatePasswordHash(ctx, userId, passwordHash)
}

// sendTelegramPasswordReset sends password reset link via Telegram Bot API
func (s *service) sendTelegramPasswordReset(telegramId string, resetURL string, locale *string) error {
	if lib.Config.TelegramToken == "" {
		return errors.New("telegram token not configured")
	}

	lang := "ru"
	if locale != nil {
		lang = *locale
	}

	var message string
	if lang == "en" {
		message = fmt.Sprintf("🔐 *Password Reset*\n\nClick the link below to reset your password:\n\n[Reset Password](%s)\n\n_This link is valid for 1 hour._", resetURL)
	} else {
		message = fmt.Sprintf("🔐 *Сброс пароля*\n\nНажмите на ссылку ниже, чтобы сбросить пароль:\n\n[Сбросить пароль](%s)\n\n_Ссылка действительна 1 час._", resetURL)
	}

	payload := map[string]interface{}{
		"chat_id":    telegramId,
		"text":       message,
		"parse_mode": "Markdown",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", lib.Config.TelegramToken)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

// sendEmailPasswordReset sends password reset link via email
func (s *service) sendEmailPasswordReset(emailAddr string, resetURL string, locale *string) error {
	lang := "ru"
	if locale != nil {
		lang = *locale
	}

	var subject, body string
	if lang == "en" {
		subject = "Nutri02 - Password Reset"
		body = fmt.Sprintf(`Hello!

You requested a password reset for your Nutri02 account.

Click the link below to reset your password:
%s

This link is valid for 1 hour.

If you didn't request this, please ignore this email.

Best regards,
Nutri02 Team`, resetURL)
	} else {
		subject = "Nutri02 - Сброс пароля"
		body = fmt.Sprintf(`Привет!

Вы запросили сброс пароля для вашего аккаунта Nutri02.

Перейдите по ссылке ниже, чтобы сбросить пароль:
%s

Ссылка действительна 1 час.

Если вы не запрашивали сброс пароля, просто проигнорируйте это письмо.

С уважением,
Команда Nutri02`, resetURL)
	}

	return s.emailService.SendRawEmail(context.Background(), emailAddr, subject, body)
}

// maskEmail masks email for privacy (e.g., j***@gmail.com)
func maskEmail(emailStr string) string {
	parts := strings.Split(emailStr, "@")
	if len(parts) != 2 {
		return "***"
	}

	local := parts[0]
	domain := parts[1]

	if len(local) <= 1 {
		return local + "***@" + domain
	}

	return string(local[0]) + "***@" + domain
}
