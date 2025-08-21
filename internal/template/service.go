package template

import (
	"context"
)

type Service interface {
	GetLikeName(ctx context.Context, name string) ([]Template, error)
}

type service struct {
	repo Repository
}

func NewService() Service {
	return &service{repo: NewRepository()}
}

func (s *service) GetLikeName(ctx context.Context, name string) ([]Template, error) {
	return s.repo.GetLikeName(context.Background(), name)
}
