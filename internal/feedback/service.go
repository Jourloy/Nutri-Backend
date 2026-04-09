package feedback

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/jourloy/somivyn/internal/database"
)

const cooldown = 7 * 24 * time.Hour
const minProductsForFeedback = 5

type Service interface {
	GetStatus(ctx context.Context, userID string) (*StatusResponse, error)
	Submit(ctx context.Context, userID, status string, message *string) (*Feedback, error)
	SetViewed(ctx context.Context, id string, viewed bool) (*Feedback, error)
}

type service struct {
	repo Repository
}

func NewService() Service {
	return &service{
		repo: NewRepository(),
	}
}

func (s *service) GetStatus(ctx context.Context, userID string) (*StatusResponse, error) {
	last, err := s.repo.GetLatestByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	resp := &StatusResponse{
		ShouldShow:    false,
		CooldownHours: int(cooldown.Hours()),
		LastFeedback:  last,
	}

	cooldownSatisfied := true

	if last != nil {
		createdAt := last.CreatedAt.UTC()
		last.CreatedAt = createdAt

		next := createdAt.Add(cooldown)
		resp.NextAllowedAt = &next
		if now.Before(next) {
			cooldownSatisfied = false
		}
	}

	activitySatisfied, activityNextAllowedAt, err := s.checkProductEligibility(ctx, userID, now)
	if err != nil {
		return nil, err
	}

	if activityNextAllowedAt != nil {
		if resp.NextAllowedAt == nil || resp.NextAllowedAt.Before(*activityNextAllowedAt) {
			resp.NextAllowedAt = activityNextAllowedAt
		}
	}

	resp.ShouldShow = cooldownSatisfied && activitySatisfied

	return resp, nil
}

func (s *service) Submit(ctx context.Context, userID, status string, message *string) (*Feedback, error) {
	var sanitizedMessage *string
	if message != nil {
		trimmed := strings.TrimSpace(*message)
		if trimmed != "" {
			msg := trimmed
			sanitizedMessage = &msg
		}
	}

	payload := Feedback{
		UserID:  userID,
		Status:  status,
		Message: sanitizedMessage,
		Viewed:  false,
	}

	return s.repo.Create(ctx, payload)
}

func (s *service) SetViewed(ctx context.Context, id string, viewed bool) (*Feedback, error) {
	return s.repo.UpdateViewed(ctx, id, viewed)
}

func (s *service) checkProductEligibility(ctx context.Context, userID string, now time.Time) (bool, *time.Time, error) {
	fifthCreatedAt, err := s.getNthProductCreatedAt(ctx, userID, minProductsForFeedback)
	if err != nil {
		return false, nil, err
	}

	if fifthCreatedAt == nil {
		return false, nil, nil
	}

	fifthUTC := fifthCreatedAt.UTC()
	nextDayStart := time.Date(fifthUTC.Year(), fifthUTC.Month(), fifthUTC.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)

	if now.Before(nextDayStart) {
		return false, &nextDayStart, nil
	}

	return true, nil, nil
}

func (s *service) getNthProductCreatedAt(ctx context.Context, userID string, n int) (*time.Time, error) {
	if n <= 0 {
		return nil, nil
	}

	const query = `
		SELECT created_at
		FROM products
		WHERE user_id = $1
		ORDER BY created_at ASC
		OFFSET $2
		LIMIT 1;
	`

	var createdAt time.Time
	err := database.Database.GetContext(ctx, &createdAt, query, userID, n-1)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &createdAt, nil
}
