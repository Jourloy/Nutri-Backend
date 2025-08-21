package fit

import (
	"context"
)

type Service interface {
	CreateFitProfile(fc FitProfileCreate) (*FitProfile, error)
	GetFitProfileByUser(uid string) (*FitProfile, error)
	GetFitProfileById(id string) (*FitProfile, error)
	UpdateFitProfile(ctx context.Context, fu FitProfileCreate, uid string) (*FitProfile, error)
}

type service struct {
	repo Repository
}

func NewService() Service {
	return &service{repo: NewRepository()}
}

func (s *service) CreateFitProfile(fc FitProfileCreate) (*FitProfile, error) {
	return s.repo.CreateFitProfile(context.Background(), fc)
}

func (s *service) GetFitProfileByUser(uid string) (*FitProfile, error) {
	return s.repo.GetFitProfileByUser(context.Background(), uid)
}

func (s *service) GetFitProfileById(id string) (*FitProfile, error) {
	return s.repo.GetFitProfileById(context.Background(), id)
}

func (s *service) UpdateFitProfile(ctx context.Context, fu FitProfileCreate, uid string) (*FitProfile, error) {
	f, err := s.repo.GetFitProfileByUser(ctx, uid)
	if err != nil {
		return nil, err
	}

	return s.repo.UpdateFitProfile(context.Background(), fu, uid, f.Id)
}
