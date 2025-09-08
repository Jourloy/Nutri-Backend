package analytics

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Service interface {
	GetSeries(ctx context.Context, userId string, end time.Time, days int) (*SeriesResponse, error)
}

type service struct{ db *sqlx.DB }

func NewService() Service { return &service{db: database.Database} }

func (s *service) GetSeries(ctx context.Context, userId string, end time.Time, days int) (*SeriesResponse, error) {
	if days <= 0 {
		days = 30
	}
	// Plan gating
	planType := s.getPlanType(ctx, userId)

	allowed := days
	clamped := false
	if planType == "START" || planType == "start" {
		if days > 7 {
			allowed = 7
			clamped = true
		}
	}

	// Compute range
	endDay := end.Truncate(24 * time.Hour)
	startDay := endDay.AddDate(0, 0, -allowed+1)

	// Query aggregates from products
	rows, err := s.db.QueryxContext(ctx, `
        SELECT created_at::date AS d,
               COALESCE(SUM(calories),0)::float,
               COALESCE(SUM(protein),0)::float,
               COALESCE(SUM(fat),0)::float,
               COALESCE(SUM(carbs),0)::float
        FROM products
        WHERE user_id=$1 AND created_at::date >= $2::date AND created_at::date <= $3::date
        GROUP BY d
        ORDER BY d`, userId, startDay, endDay)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	agg := map[string]Day{}
	for rows.Next() {
		var d Day
		if err := rows.Scan(&d.Date, &d.Calories, &d.Protein, &d.Fat, &d.Carbs); err != nil {
			return nil, err
		}
		key := d.Date.Format("2006-01-02")
		agg[key] = d
	}
	// fill missing days
	res := make([]Day, 0, allowed)
	for i := 0; i < allowed; i++ {
		day := startDay.AddDate(0, 0, i)
		key := day.Format("2006-01-02")
		if v, ok := agg[key]; ok {
			res = append(res, v)
		} else {
			res = append(res, Day{Date: day})
		}
	}

	return &SeriesResponse{
		Days:        res,
		AllowedDays: allowed,
		Clamped:     clamped,
		PlanType:    planType,
		RangeStart:  startDay.Format("2006-01-02"),
		RangeEnd:    endDay.Format("2006-01-02"),
	}, nil
}

func (s *service) getPlanType(ctx context.Context, userId string) string {
	// Latest subscription plan type
	var planType string
	_ = s.db.GetContext(ctx, &planType, `
        SELECT p.code
        FROM subscriptions s
        JOIN plans p ON p.id = s.plan_id
        WHERE s.user_id = $1
        ORDER BY s.created_at DESC
        LIMIT 1`, userId)
	if planType == "" {
		return "START"
	}
	return planType
}
