package product

import (
	"context"

	"github.com/jourloy/nutri-backend/internal/fit"
)

type Service interface {
	CreateProduct(ctx context.Context, pc ProductCreate) (*Product, error)
	GetAll(ctx context.Context, uid string) ([]Product, error)
	GetAllByToday(ctx context.Context, uid string) ([]Product, error)
	GetLikeName(ctx context.Context, name string, uid string) ([]Product, error)
	UpdateProduct(ctx context.Context, pu Product, uid string) (*Product, error)
	DeleteProduct(ctx context.Context, id int64, uid string) error
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

	return s.repo.CreateProduct(ctx, pc)
}

func (s *service) GetAll(ctx context.Context, uid string) ([]Product, error) {
	f, err := s.fitService.GetFitProfileByUser(uid)
	if err != nil {
		return nil, err
	}

	return s.repo.GetAll(ctx, f.Id, uid)
}

func (s *service) GetAllByToday(ctx context.Context, uid string) ([]Product, error) {
	f, err := s.fitService.GetFitProfileByUser(uid)
	if err != nil {
		return nil, err
	}

	return s.repo.GetAllByToday(ctx, f.Id, uid)
}

func (s *service) GetLikeName(ctx context.Context, name string, uid string) ([]Product, error) {
	f, err := s.fitService.GetFitProfileByUser(uid)
	if err != nil {
		return nil, err
	}

	return s.repo.GetLikeName(ctx, name, f.Id, uid)
}

func (s *service) UpdateProduct(ctx context.Context, pu Product, uid string) (*Product, error) {
	f, err := s.fitService.GetFitProfileByUser(uid)
	if err != nil {
		return nil, err
	}

	return s.repo.UpdateProduct(ctx, pu, f.Id, uid)
}

func (s *service) DeleteProduct(ctx context.Context, id int64, uid string) error {
	f, err := s.fitService.GetFitProfileByUser(uid)
	if err != nil {
		return err
	}

	return s.repo.DeleteProduct(ctx, id, f.Id, uid)
}
