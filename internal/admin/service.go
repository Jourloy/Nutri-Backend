package admin

import (
	"context"
	"database/sql"

	"github.com/jourloy/nutri02/internal/user"
)

type Service interface {
	// Dashboard
	GetDashboardStats(ctx context.Context) (*DashboardStats, error)

	// Users
	GetAllUsers(ctx context.Context, limit, offset int, sortBy UserSortBy, sortOrder SortOrder) ([]UserListItem, int64, error)
	GetUserDetails(ctx context.Context, userId string) (*UserDetailsResponse, error)
	CreateUserWithSubscription(ctx context.Context, username, passwordHash, email string, planId int64, durationMs int64) error
	GrantUserSubscription(ctx context.Context, userId string, planId, durationDays int64) (*AdminUserSubscription, error)
	DeleteUser(ctx context.Context, userId string) error

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
	repo     Repository
	userRepo user.Repository
}

func NewService() Service {
	return &service{
		repo:     NewRepository(),
		userRepo: user.NewRepository(),
	}
}

func (s *service) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	return s.repo.GetDashboardStats(ctx)
}

func (s *service) GetAllUsers(ctx context.Context, limit, offset int, sortBy UserSortBy, sortOrder SortOrder) ([]UserListItem, int64, error) {
	users, err := s.repo.GetAllUsers(ctx, limit, offset, sortBy, sortOrder)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.GetUserCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return users, count, nil
}

func (s *service) GetUserDetails(ctx context.Context, userId string) (*UserDetailsResponse, error) {
	return s.repo.GetUserDetails(ctx, userId)
}

func (s *service) CreateUserWithSubscription(ctx context.Context, username, passwordHash, email string, planId int64, durationMs int64) error {
	return s.repo.CreateUserWithSubscription(ctx, username, passwordHash, email, planId, durationMs)
}

func (s *service) GrantUserSubscription(ctx context.Context, userId string, planId, durationDays int64) (*AdminUserSubscription, error) {
	return s.repo.GrantUserSubscription(ctx, userId, planId, durationDays)
}

func (s *service) DeleteUser(ctx context.Context, userId string) error {
	deleted, err := s.userRepo.DeleteUser(ctx, userId)
	if err != nil {
		return err
	}
	if deleted == nil {
		return sql.ErrNoRows
	}
	return nil
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
