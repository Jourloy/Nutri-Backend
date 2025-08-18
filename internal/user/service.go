package user

import (
	"context"
)

type Service interface {
	CreateUser(user *UserCreate) (*User, error)
	GetUser(id string) (*User, error)
	GetUserByUsername(username string) (*User, error)
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

func (s *service) DeleteUser(ctx context.Context, id string) (*User, error) {
	logger.Debug("Delete in user service")
	return s.repo.DeleteUser(context.Background(), id)
}
