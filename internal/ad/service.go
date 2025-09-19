package ad

import (
    "context"

    "github.com/jmoiron/sqlx"

    "github.com/jourloy/nutri-backend/internal/database"
)

type Service interface {
    TrackLanding(ctx context.Context, code, ip, ua, referer string) error
}

type service struct{ db *sqlx.DB }

func NewService() Service { return &service{db: database.Database} }

func (s *service) TrackLanding(ctx context.Context, code, ip, ua, referer string) error {
    if code == "" {
        return nil
    }
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO landing_hits (code, ip, user_agent, referer)
        VALUES ($1, $2, $3, $4)
    `, code, ip, ua, referer)
    return err
}

