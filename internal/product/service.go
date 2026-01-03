package product

import (
	"context"
	"errors"

	"github.com/jourloy/nutri-backend/internal/fit"
)

type Service interface {
	CreateProduct(ctx context.Context, pc ProductCreate, today string) ([]Product, error)
	GetAll(ctx context.Context, uid string) ([]Product, error)
	GetAllByToday(ctx context.Context, uid string, today string) ([]Product, error)
	GetAllByDate(ctx context.Context, date string, uid string) ([]Product, error)
	GetLikeName(ctx context.Context, name string, uid string) ([]Product, error)
	UpdateProduct(ctx context.Context, pu Product, uid string) (*Product, error)
	DeleteProduct(ctx context.Context, id int64, uid string, today string) ([]Product, error)
}

type service struct {
	repo       Repository
	fitService fit.Service
}

func NewService() Service {
	return &service{repo: NewRepository(), fitService: fit.NewService()}
}

func (s *service) CreateProduct(ctx context.Context, pc ProductCreate, today string) ([]Product, error) {
	f, err := s.fitService.GetFitProfileByUser(pc.UserId)
	if err != nil {
		return nil, err
	}
	pc.FitId = f.Id

	// Validate meal_type (only if provided)
	if pc.MealType != nil {
		validMealTypes := map[string]bool{
			"breakfast": true,
			"lunch":     true,
			"dinner":    true,
			"snack":     true,
		}
		if !validMealTypes[*pc.MealType] {
			pc.MealType = nil // Set to nil if invalid
		}
	}

	count, err := s.repo.GetCountByToday(ctx, f.Id, pc.UserId, today)
	if err != nil {
		return nil, err
	}
	if count >= 20 {
		return nil, errors.New("you have reached the maximum number of products for today")
	}

	return s.repo.CreateProduct(ctx, pc, today)
}

func (s *service) GetAll(ctx context.Context, uid string) ([]Product, error) {
	f, err := s.fitService.GetFitProfileByUser(uid)
	if err != nil {
		return nil, err
	}

	return s.repo.GetAll(ctx, f.Id, uid)
}

func (s *service) GetAllByToday(ctx context.Context, uid string, today string) ([]Product, error) {
	f, err := s.fitService.GetFitProfileByUser(uid)
	if err != nil {
		return nil, err
	}

	return s.repo.GetAllByToday(ctx, f.Id, uid, today)
}

func (s *service) GetAllByDate(ctx context.Context, date string, uid string) ([]Product, error) {
	f, err := s.fitService.GetFitProfileByUser(uid)
	if err != nil {
		return nil, err
	}

	return s.repo.GetAllByDate(ctx, date, f.Id, uid)
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

	// Validate meal_type (only if provided)
	if pu.MealType != nil {
		validMealTypes := map[string]bool{
			"breakfast": true,
			"lunch":     true,
			"dinner":    true,
			"snack":     true,
		}
		if !validMealTypes[*pu.MealType] {
			pu.MealType = nil // Set to nil if invalid
		}
	}

	return s.repo.UpdateProduct(ctx, pu, f.Id, uid)
}

func (s *service) DeleteProduct(ctx context.Context, id int64, uid string, today string) ([]Product, error) {
	f, err := s.fitService.GetFitProfileByUser(uid)
	if err != nil {
		return nil, err
	}

	return s.repo.DeleteProduct(ctx, id, f.Id, uid, today)
}
