package product

import (
	"context"

	"github.com/jourloy/nutri-backend/internal/fit"
)

type Service interface {
	CreateProduct(ctx context.Context, pc ProductCreate) (*Product, error)
	GetAll(ctx context.Context, uid string) ([]Product, error)
	GetAllByToday(ctx context.Context, uid string) ([]Product, error)
}

type service struct {
	repo       Repository
	fitService fit.Service
}

func NewService() Service {
	return &service{repo: NewRepository(), fitService: fit.NewService()}
}

func (s *service) CreateProduct(ctx context.Context, pc ProductCreate) (*Product, error) {
	f, err := s.fitService.GetFitProfileByUser(pc.UserId)
	if err != nil {
		return nil, err
	}
	pc.FitId = f.Id

	return s.repo.CreateProduct(context.Background(), pc)
}

func (s *service) GetAll(ctx context.Context, uid string) ([]Product, error) {
	f, err := s.fitService.GetFitProfileByUser(uid)
	if err != nil {
		return nil, err
	}

	return s.repo.GetAll(context.Background(), f.Id, uid)
}

func (s *service) GetAllByToday(ctx context.Context, uid string) ([]Product, error) {
	f, err := s.fitService.GetFitProfileByUser(uid)
	if err != nil {
		return nil, err
	}

	return s.repo.GetAllByToday(context.Background(), f.Id, uid)
}
