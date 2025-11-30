package email

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"os"
	"time"

	"github.com/mailgun/mailgun-go/v4"
)

var (
	ErrCodeExpired        = errors.New("verification code expired")
	ErrCodeNotFound       = errors.New("verification code not found")
	ErrInvalidCode        = errors.New("invalid verification code")
	ErrEmailConfigMissing = errors.New("mailgun configuration missing")
)

// UserLocaleGetter интерфейс для получения локали пользователя
type UserLocaleGetter interface {
	GetUserLocale(ctx context.Context, userId string) (string, error)
}

type Service interface {
	SendVerificationCode(ctx context.Context, userId string, email string) error
	VerifyCode(ctx context.Context, userId string, code string) (*VerificationCode, error)
	ResendVerificationCode(ctx context.Context, userId string) error
	SetUserLocaleGetter(getter UserLocaleGetter)
}

type service struct {
	repo             Repository
	templates        *EmailTemplates
	userLocaleGetter UserLocaleGetter
	mailgunDomain    string
	mailgunAPIKey    string
	mailgunFromEmail string
}

func NewService() Service {
	return &service{
		repo:             NewRepository(),
		templates:        NewEmailTemplates(),
		mailgunDomain:    os.Getenv("MAILGUN_DOMAIN"),
		mailgunAPIKey:    os.Getenv("MAILGUN_API_KEY"),
		mailgunFromEmail: os.Getenv("MAILGUN_FROM_EMAIL"),
	}
}

// SetUserLocaleGetter устанавливает getter для получения локали пользователя
func (s *service) SetUserLocaleGetter(getter UserLocaleGetter) {
	s.userLocaleGetter = getter
}

// generateVerificationCode генерирует 6-значный код
func (s *service) generateVerificationCode() (string, error) {
	const digits = "0123456789"
	code := make([]byte, 6)

	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[num.Int64()]
	}

	return string(code), nil
}

// sendEmailViaMailgun отправляет email через Mailgun
func (s *service) sendEmailViaMailgun(to string, subject string, body string) error {
	if s.mailgunDomain == "" || s.mailgunAPIKey == "" || s.mailgunFromEmail == "" {
		return ErrEmailConfigMissing
	}

	mg := mailgun.NewMailgun(s.mailgunDomain, s.mailgunAPIKey)
	mg.SetAPIBase(mailgun.APIBaseUS) // Используем EU endpoint, можно настроить через env

	message := mg.NewMessage(
		s.mailgunFromEmail,
		subject,
		body,
		to,
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, _, err := mg.Send(ctx, message)
	return err
}

// SendVerificationCode создает и отправляет код верификации
func (s *service) SendVerificationCode(ctx context.Context, userId string, email string) error {
	// Генерируем код
	code, err := s.generateVerificationCode()
	if err != nil {
		return err
	}

	// Создаем запись в БД (код действителен 15 минут)
	codeCreate := &VerificationCodeCreate{
		UserId:    userId,
		Email:     email,
		Code:      code,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}

	_, err = s.repo.CreateVerificationCode(ctx, codeCreate)
	if err != nil {
		return err
	}

	// Получаем локаль пользователя
	locale := "ru" // Default locale
	if s.userLocaleGetter != nil {
		userLocale, err := s.userLocaleGetter.GetUserLocale(ctx, userId)
		if err == nil && userLocale != "" {
			locale = userLocale
		}
	}

	// Получаем локализованный шаблон
	subject, body := s.templates.GetVerificationCodeTemplate(locale, code)

	return s.sendEmailViaMailgun(email, subject, body)
}

// VerifyCode проверяет код верификации
func (s *service) VerifyCode(ctx context.Context, userId string, code string) (*VerificationCode, error) {
	// Получаем код из БД
	vc, err := s.repo.GetVerificationCode(ctx, userId, code)
	if err != nil {
		return nil, ErrCodeNotFound
	}

	// Проверяем, не истек ли код
	if time.Now().After(vc.ExpiresAt) {
		return nil, ErrCodeExpired
	}

	// Помечаем код как использованный
	verifiedCode, err := s.repo.MarkAsVerified(ctx, vc.Id)
	if err != nil {
		return nil, err
	}

	// Очищаем истекшие коды (асинхронно, не ждем результата)
	go s.repo.DeleteExpiredCodes(context.Background())

	return verifiedCode, nil
}

// ResendVerificationCode повторно отправляет код верификации
func (s *service) ResendVerificationCode(ctx context.Context, userId string) error {
	// Получаем последний код
	vc, err := s.repo.GetLatestVerificationCode(ctx, userId)
	if err != nil {
		return ErrCodeNotFound
	}

	// Проверяем, прошло ли 60 секунд с момента создания
	if time.Now().Before(vc.CreatedAt.Add(60 * time.Second)) {
		return errors.New("please wait before requesting a new code")
	}

	// Отправляем новый код
	return s.SendVerificationCode(ctx, userId, vc.Email)
}
