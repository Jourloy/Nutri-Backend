package user

import (
	"context"
)

type Service interface {
	CreateUser(user *UserCreate) (*User, error)
	GetUser(id string) (*User, error)
	GetUserByUsername(username string) (*User, error)
	IncreaseViewUpdates(ctx context.Context, uid string) (*User, error)
	UpdateLogin(ctx context.Context, uid string) error
	DeleteUser(ctx context.Context, id string) (*User, error)
}

type service struct {
	repo Repository
}

func NewService() Service {
	return &service{repo: NewRepository()}
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

func (s *service) IncreaseViewUpdates(ctx context.Context, uid string) (*User, error) {
	return s.repo.IncreaseViewUpdates(ctx, uid)
}

func (s *service) UpdateLogin(ctx context.Context, uid string) error {
	return s.repo.UpdateLogin(ctx, uid)
}

func (s *service) DeleteUser(ctx context.Context, id string) (*User, error) {
	return s.repo.DeleteUser(ctx, id)
}
