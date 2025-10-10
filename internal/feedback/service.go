package feedback

import (
	"context"
	"strings"
	"time"
)

const cooldown = 7 * 24 * time.Hour

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
		ShouldShow:    true,
		CooldownHours: int(cooldown.Hours()),
		LastFeedback:  last,
	}

	if last != nil {
		createdAt := last.CreatedAt.UTC()
		last.CreatedAt = createdAt

		next := createdAt.Add(cooldown)
		resp.NextAllowedAt = &next
		if now.Before(next) {
			resp.ShouldShow = false
		}
	}

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
