package admin

import (
	"context"
)

type Service interface {
	// Dashboard
	GetDashboardStats(ctx context.Context) (*DashboardStats, error)

	// Users
	GetAllUsers(ctx context.Context, limit, offset int) ([]UserListItem, int64, error)
	CreateUserWithSubscription(ctx context.Context, username, passwordHash, email string, planId int64, durationMs int64) error

	// Plans
	UpdatePlanPrice(ctx context.Context, planId int64, amountMinor int64) error
	UpdateUserSubscriptionPrice(ctx context.Context, userId string, amountMinor int64) error
	UpdatePlanFeatures(ctx context.Context, planId int64, features map[string]interface{}) error

	// Notifications
	CreateNotification(ctx context.Context, createdBy string, notification *AdminNotificationCreate) (*AdminNotification, error)
	GetNotifications(ctx context.Context, limit, offset int) ([]AdminNotification, error)
	SendNotification(ctx context.Context, notificationId int64) error
}

type service struct {
	repo Repository
}

func NewService() Service {
	return &service{repo: NewRepository()}
}

func (s *service) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	return s.repo.GetDashboardStats(ctx)
}

func (s *service) GetAllUsers(ctx context.Context, limit, offset int) ([]UserListItem, int64, error) {
	users, err := s.repo.GetAllUsers(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.GetUserCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return users, count, nil
}

func (s *service) CreateUserWithSubscription(ctx context.Context, username, passwordHash, email string, planId int64, durationMs int64) error {
	return s.repo.CreateUserWithSubscription(ctx, username, passwordHash, email, planId, durationMs)
}

func (s *service) UpdatePlanPrice(ctx context.Context, planId int64, amountMinor int64) error {
	return s.repo.UpdatePlanPrice(ctx, planId, amountMinor)
}

func (s *service) UpdateUserSubscriptionPrice(ctx context.Context, userId string, amountMinor int64) error {
	return s.repo.UpdateUserSubscriptionPrice(ctx, userId, amountMinor)
}

func (s *service) UpdatePlanFeatures(ctx context.Context, planId int64, features map[string]interface{}) error {
	return s.repo.UpdatePlanFeatures(ctx, planId, features)
}

func (s *service) CreateNotification(ctx context.Context, createdBy string, notification *AdminNotificationCreate) (*AdminNotification, error) {
	return s.repo.CreateNotification(ctx, createdBy, notification)
}

func (s *service) GetNotifications(ctx context.Context, limit, offset int) ([]AdminNotification, error) {
	return s.repo.GetNotifications(ctx, limit, offset)
}

func (s *service) SendNotification(ctx context.Context, notificationId int64) error {
	return s.repo.SendNotification(ctx, notificationId)
}
