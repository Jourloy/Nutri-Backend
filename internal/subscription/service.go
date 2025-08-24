package subscription

import "context"

type Service interface {
	Create(ctx context.Context, sc SubscriptionCreate) (*Subscription, error)
	Update(ctx context.Context, s Subscription) (*Subscription, error)
	Delete(ctx context.Context, id int64, uid string) error
}

type service struct {
	repo Repository
}

func NewService() Service {
	return &service{repo: NewRepository()}
}

func (s *service) Create(ctx context.Context, sc SubscriptionCreate) (*Subscription, error) {
	return s.repo.Create(ctx, sc)
}

func (s *service) Update(ctx context.Context, sub Subscription) (*Subscription, error) {
	return s.repo.Update(ctx, sub)
}

func (s *service) Delete(ctx context.Context, id int64, uid string) error {
	return s.repo.Delete(ctx, id, uid)
}
