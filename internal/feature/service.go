package feature

import "context"

type Service interface {
	Create(ctx context.Context, f Feature) (*Feature, error)
	Update(ctx context.Context, f Feature) (*Feature, error)
	Delete(ctx context.Context, key string) error
}

type service struct {
	repo Repository
}

func NewService() Service {
	return &service{repo: NewRepository()}
}

func (s *service) Create(ctx context.Context, f Feature) (*Feature, error) {
	return s.repo.Create(ctx, f)
}

func (s *service) Update(ctx context.Context, f Feature) (*Feature, error) {
	return s.repo.Update(ctx, f)
}

func (s *service) Delete(ctx context.Context, key string) error {
	return s.repo.Delete(ctx, key)
}
