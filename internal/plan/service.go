package plan

import "context"

type Service interface {
	GetAllActive(ctx context.Context) ([]Plan, error)
	Create(ctx context.Context, pc PlanCreate) (*Plan, error)
	Update(ctx context.Context, p Plan) (*Plan, error)
	Delete(ctx context.Context, id int64) error
}

type service struct {
	repo Repository
}

func NewService() Service {
	return &service{repo: NewRepository()}
}

func (s *service) GetAllActive(ctx context.Context) ([]Plan, error) {
	return s.repo.GetAllActive(ctx)
}

func (s *service) Create(ctx context.Context, pc PlanCreate) (*Plan, error) {
	return s.repo.Create(ctx, pc)
}

func (s *service) Update(ctx context.Context, p Plan) (*Plan, error) {
	return s.repo.Update(ctx, p)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
