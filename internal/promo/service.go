package promo

import "context"

type Service interface {
	CreatePromoCode(ctx context.Context, createdBy string, promo *PromoCodeCreate) (*PromoCode, error)
	GetPromoCode(ctx context.Context, code string) (*PromoCode, error)
	GetAllPromoCodes(ctx context.Context, limit, offset int) ([]PromoCode, error)
	UpdatePromoCode(ctx context.Context, id int64, promo *PromoCodeCreate) (*PromoCode, error)
	DeletePromoCode(ctx context.Context, id int64) error
	ValidatePromoCode(ctx context.Context, req *ValidatePromoRequest) (*ValidatePromoResponse, error)
	UsePromoCode(ctx context.Context, id int64) error
}

type service struct {
	repo Repository
}

func NewService() Service {
	return &service{repo: NewRepository()}
}

func (s *service) CreatePromoCode(ctx context.Context, createdBy string, promo *PromoCodeCreate) (*PromoCode, error) {
	return s.repo.CreatePromoCode(ctx, createdBy, promo)
}

func (s *service) GetPromoCode(ctx context.Context, code string) (*PromoCode, error) {
	return s.repo.GetPromoCode(ctx, code)
}

func (s *service) GetAllPromoCodes(ctx context.Context, limit, offset int) ([]PromoCode, error) {
	return s.repo.GetAllPromoCodes(ctx, limit, offset)
}

func (s *service) UpdatePromoCode(ctx context.Context, id int64, promo *PromoCodeCreate) (*PromoCode, error) {
	return s.repo.UpdatePromoCode(ctx, id, promo)
}

func (s *service) DeletePromoCode(ctx context.Context, id int64) error {
	return s.repo.DeletePromoCode(ctx, id)
}

func (s *service) ValidatePromoCode(ctx context.Context, req *ValidatePromoRequest) (*ValidatePromoResponse, error) {
	return s.repo.ValidateAndCalculate(ctx, req.Code, req.PlanId, req.AmountMinor)
}

func (s *service) UsePromoCode(ctx context.Context, id int64) error {
	return s.repo.IncrementUsage(ctx, id)
}
