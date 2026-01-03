package user

import (
	"context"

	"github.com/jourloy/nutri-backend/internal/email"
)

type Service interface {
	CreateUser(user *UserCreate) (*User, error)
	GetUser(id string) (*User, error)
	GetUserByUsername(username string) (*User, error)
	GetUserLocale(ctx context.Context, uid string) (string, error)
	IncreaseViewUpdates(ctx context.Context, uid string) (*User, error)
	UpdateLogin(ctx context.Context, uid string) error
	DeleteUser(ctx context.Context, id string) (*User, error)
	InvalidateTokens(ctx context.Context, id string) error
	UpdateLocale(ctx context.Context, uid string, locale string) (*User, error)
	UpdateTimezone(ctx context.Context, uid string, timezone string) (*User, error)
	GetUserStats(ctx context.Context, uid string) (*UserStats, error)
	RequestEmailVerification(ctx context.Context, uid string, email string) error
	VerifyEmail(ctx context.Context, uid string, code string) (*User, error)
	ResendVerificationCode(ctx context.Context, uid string) error
}

type service struct {
	repo         Repository
	emailService email.Service
}

func NewService() Service {
	s := &service{
		repo:         NewRepository(),
		emailService: email.NewService(),
	}
	// Устанавливаем locale getter для email service
	s.emailService.SetUserLocaleGetter(s)
	return s
}

func (s *service) CreateUser(userCreate *UserCreate) (*User, error) {
	return s.repo.CreateUser(context.Background(), userCreate)
}

func (s *service) GetUser(id string) (*User, error) {
	return s.repo.GetUser(context.Background(), id)
}

func (s *service) GetUserByUsername(username string) (*User, error) {
	return s.repo.GetUserByUsername(context.Background(), username)
}

func (s *service) GetUserLocale(ctx context.Context, uid string) (string, error) {
	user, err := s.repo.GetUser(ctx, uid)
	if err != nil {
		return "", err
	}
	return *user.Locale, nil
}

func (s *service) IncreaseViewUpdates(ctx context.Context, uid string) (*User, error) {
	return s.repo.IncreaseViewUpdates(ctx, uid)
}

func (s *service) UpdateLogin(ctx context.Context, uid string) error {
	return s.repo.UpdateLogin(ctx, uid)
}

func (s *service) DeleteUser(ctx context.Context, id string) (*User, error) {
	return s.repo.DeleteUser(ctx, id)
}

func (s *service) InvalidateTokens(ctx context.Context, id string) error {
	return s.repo.InvalidateTokens(ctx, id)
}

func (s *service) UpdateLocale(ctx context.Context, uid string, locale string) (*User, error) {
	return s.repo.UpdateLocale(ctx, uid, locale)
}

func (s *service) UpdateTimezone(ctx context.Context, uid string, timezone string) (*User, error) {
	return s.repo.UpdateTimezone(ctx, uid, timezone)
}

func (s *service) GetUserStats(ctx context.Context, uid string) (*UserStats, error) {
	return s.repo.GetUserStats(ctx, uid)
}

func (s *service) RequestEmailVerification(ctx context.Context, uid string, email string) error {
	// Отправляем код верификации через email service
	return s.emailService.SendVerificationCode(ctx, uid, email)
}

func (s *service) VerifyEmail(ctx context.Context, uid string, code string) (*User, error) {
	// Проверяем код через email service
	verifiedCode, err := s.emailService.VerifyCode(ctx, uid, code)
	if err != nil {
		return nil, err
	}

	// Обновляем user - устанавливаем email и email_verified = true
	return s.repo.SetEmailVerified(ctx, uid, verifiedCode.Email)
}

func (s *service) ResendVerificationCode(ctx context.Context, uid string) error {
	// Повторно отправляем код через email service
	return s.emailService.ResendVerificationCode(ctx, uid)
}
