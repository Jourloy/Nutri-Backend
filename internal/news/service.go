package news

import (
	"context"
)

type Service interface {
	// Admin operations
	CreateNews(ctx context.Context, news NewsCreate) (*News, error)
	UpdateNews(ctx context.Context, news NewsUpdate) (*News, error)
	DeleteNews(ctx context.Context, id int64) error
	GetNewsById(ctx context.Context, id int64) (*News, error)
	GetAllNews(ctx context.Context, includeUnpublished bool) ([]News, error)
	PublishNews(ctx context.Context, id int64) error
	UnpublishNews(ctx context.Context, id int64) error
	UpdateNewsStatus(ctx context.Context, id int64, status string) error
	GetPreviewNews(ctx context.Context, limit int) ([]News, error)

	// User operations
	GetPublishedNews(ctx context.Context, userId string, limit int) (*NewsListResponse, error)
	MarkAsViewed(ctx context.Context, userId string, newsId int64) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// ===== Admin Operations =====

func (s *service) CreateNews(ctx context.Context, news NewsCreate) (*News, error) {
	return s.repo.Create(ctx, news)
}

func (s *service) UpdateNews(ctx context.Context, news NewsUpdate) (*News, error) {
	return s.repo.Update(ctx, news)
}

func (s *service) DeleteNews(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) GetNewsById(ctx context.Context, id int64) (*News, error) {
	return s.repo.GetById(ctx, id)
}

func (s *service) GetAllNews(ctx context.Context, includeUnpublished bool) ([]News, error) {
	return s.repo.GetAll(ctx, includeUnpublished)
}

func (s *service) PublishNews(ctx context.Context, id int64) error {
	return s.repo.Publish(ctx, id)
}

func (s *service) UnpublishNews(ctx context.Context, id int64) error {
	return s.repo.Unpublish(ctx, id)
}

func (s *service) UpdateNewsStatus(ctx context.Context, id int64, status string) error {
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *service) GetPreviewNews(ctx context.Context, limit int) ([]News, error) {
	return s.repo.GetPreview(ctx, limit)
}

// ===== User Operations =====

func (s *service) GetPublishedNews(ctx context.Context, userId string, limit int) (*NewsListResponse, error) {
	// Get published news
	newsList, err := s.repo.GetPublished(ctx, limit)
	if err != nil {
		return nil, err
	}

	// Get user's last viewed news ID
	lastViewedId, err := s.repo.GetUserLastViewedNewsId(ctx, userId)
	if err != nil {
		// If error, just set to nil (user might not have viewed any news)
		lastViewedId = nil
	}

	// Calculate unread count
	unreadCount, err := s.repo.GetUnreadCount(ctx, userId)
	if err != nil {
		unreadCount = 0
	}

	return &NewsListResponse{
		News:         newsList,
		UnreadCount:  unreadCount,
		LastViewedId: lastViewedId,
	}, nil
}

func (s *service) MarkAsViewed(ctx context.Context, userId string, newsId int64) error {
	return s.repo.UpdateUserLastViewedNews(ctx, userId, newsId)
}
