package telegram

import "context"

type Service interface {
    CreateTicket(ctx context.Context, userId string) (*TelegramProfile, error)
    LinkByToken(ctx context.Context, in LinkRequest) (*TelegramProfile, error)
    GetByUserId(ctx context.Context, userId string) (*TelegramProfile, error)
    GetPublicByUserId(ctx context.Context, userId string) (*TelegramPublic, error)
    DeleteByUserId(ctx context.Context, userId string) error
    UpdateNotifyByUserId(ctx context.Context, userId string, upd NotifyUpdate) (*TelegramProfile, error)
}

type service struct {
    repo Repository
}

func NewService() Service {
    return &service{repo: NewRepository()}
}

func (s *service) CreateTicket(ctx context.Context, userId string) (*TelegramProfile, error) {
    return s.repo.CreateOrGetTicket(ctx, userId)
}

func (s *service) LinkByToken(ctx context.Context, in LinkRequest) (*TelegramProfile, error) {
    return s.repo.LinkByToken(ctx, in)
}

func (s *service) GetByUserId(ctx context.Context, userId string) (*TelegramProfile, error) {
    return s.repo.GetByUserId(ctx, userId)
}

func (s *service) GetPublicByUserId(ctx context.Context, userId string) (*TelegramPublic, error) {
    return s.repo.GetPublicByUserId(ctx, userId)
}

func (s *service) DeleteByUserId(ctx context.Context, userId string) error {
    return s.repo.DeleteByUserId(ctx, userId)
}

func (s *service) UpdateNotifyByUserId(ctx context.Context, userId string, upd NotifyUpdate) (*TelegramProfile, error) {
    return s.repo.UpdateNotifyByUserId(ctx, userId, upd)
}
