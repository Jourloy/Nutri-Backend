package news

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/jourloy/somivyn/internal/database"
)

type Repository interface {
	// News CRUD
	Create(ctx context.Context, news NewsCreate) (*News, error)
	Update(ctx context.Context, news NewsUpdate) (*News, error)
	Delete(ctx context.Context, id int64) error
	GetById(ctx context.Context, id int64) (*News, error)
	GetAll(ctx context.Context, includeUnpublished bool) ([]News, error)
	GetPublished(ctx context.Context, limit int) ([]News, error)
	GetPreview(ctx context.Context, limit int) ([]News, error)
	Publish(ctx context.Context, id int64) error
	Unpublish(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status string) error

	// User tracking
	GetUserLastViewedNewsId(ctx context.Context, userId string) (*int64, error)
	UpdateUserLastViewedNews(ctx context.Context, userId string, newsId int64) error
	GetUnreadCount(ctx context.Context, userId string) (int, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

// ===== News CRUD =====

func (r *repository) Create(ctx context.Context, newsCreate NewsCreate) (*News, error) {
	const q = `
		INSERT INTO news (title_ru, title_en, content_ru, content_en, image_url, priority)
		VALUES (:title_ru, :title_en, :content_ru, :content_en, :image_url, :priority)
		RETURNING id, title_ru, title_en, content_ru, content_en, image_url, status,
		          is_published, published_at, priority, created_at, updated_at, deleted_at`

	rows, err := r.db.NamedQueryContext(ctx, q, newsCreate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var news News
		if err := rows.StructScan(&news); err != nil {
			return nil, err
		}
		return &news, nil
	}
	return nil, nil
}

func (r *repository) Update(ctx context.Context, newsUpdate NewsUpdate) (*News, error) {
	const q = `
		UPDATE news SET
			title_ru = :title_ru,
			title_en = :title_en,
			content_ru = :content_ru,
			content_en = :content_en,
			image_url = :image_url,
			priority = :priority,
			updated_at = NOW()
		WHERE id = :id AND deleted_at IS NULL
		RETURNING id, title_ru, title_en, content_ru, content_en, image_url, status,
		          is_published, published_at, priority, created_at, updated_at, deleted_at`

	rows, err := r.db.NamedQueryContext(ctx, q, newsUpdate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var news News
		if err := rows.StructScan(&news); err != nil {
			return nil, err
		}
		return &news, nil
	}
	return nil, nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	const q = `UPDATE news SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *repository) GetById(ctx context.Context, id int64) (*News, error) {
	const q = `
		SELECT id, title_ru, title_en, content_ru, content_en, image_url, status,
		       is_published, published_at, priority, created_at, updated_at, deleted_at
		FROM news
		WHERE id = $1 AND deleted_at IS NULL`

	var news News
	if err := r.db.GetContext(ctx, &news, q, id); err != nil {
		return nil, err
	}
	return &news, nil
}

func (r *repository) GetAll(ctx context.Context, includeUnpublished bool) ([]News, error) {
	var q string
	if includeUnpublished {
		q = `
			SELECT id, title_ru, title_en, content_ru, content_en, image_url, status,
			       is_published, published_at, priority, created_at, updated_at, deleted_at
			FROM news
			WHERE deleted_at IS NULL
			ORDER BY priority DESC, created_at DESC`
	} else {
		q = `
			SELECT id, title_ru, title_en, content_ru, content_en, image_url, status,
			       is_published, published_at, priority, created_at, updated_at, deleted_at
			FROM news
			WHERE deleted_at IS NULL AND status = 'published'
			ORDER BY priority DESC, published_at DESC`
	}

	var newsList []News
	if err := r.db.SelectContext(ctx, &newsList, q); err != nil {
		return nil, err
	}
	return newsList, nil
}

func (r *repository) GetPublished(ctx context.Context, limit int) ([]News, error) {
	const q = `
		SELECT id, title_ru, title_en, content_ru, content_en, image_url, status,
		       is_published, published_at, priority, created_at, updated_at, deleted_at
		FROM news
		WHERE deleted_at IS NULL AND status = 'published'
		ORDER BY priority DESC, published_at DESC
		LIMIT $1`

	var newsList []News
	if err := r.db.SelectContext(ctx, &newsList, q, limit); err != nil {
		return nil, err
	}
	return newsList, nil
}

func (r *repository) GetPreview(ctx context.Context, limit int) ([]News, error) {
	const q = `
		SELECT id, title_ru, title_en, content_ru, content_en, image_url, status,
		       is_published, published_at, priority, created_at, updated_at, deleted_at
		FROM news
		WHERE deleted_at IS NULL AND status = 'preview'
		ORDER BY priority DESC, created_at DESC
		LIMIT $1`

	var newsList []News
	if err := r.db.SelectContext(ctx, &newsList, q, limit); err != nil {
		return nil, err
	}
	return newsList, nil
}

func (r *repository) Publish(ctx context.Context, id int64) error {
	const q = `
		UPDATE news
		SET is_published = true, status = 'published', published_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *repository) Unpublish(ctx context.Context, id int64) error {
	const q = `
		UPDATE news
		SET is_published = false, status = 'draft', updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *repository) UpdateStatus(ctx context.Context, id int64, status string) error {
	const q = `
		UPDATE news
		SET status = $1::text,
		    is_published = CASE WHEN $1::text = 'published' THEN true ELSE false END,
		    published_at = CASE WHEN $1::text = 'published' AND published_at IS NULL THEN NOW() ELSE published_at END,
		    updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL`

	_, err := r.db.ExecContext(ctx, q, status, id)
	return err
}

// ===== User Tracking =====

func (r *repository) GetUserLastViewedNewsId(ctx context.Context, userId string) (*int64, error) {
	const q = `SELECT last_viewed_news_id FROM users WHERE id = $1`

	var newsId *int64
	if err := r.db.GetContext(ctx, &newsId, q, userId); err != nil {
		return nil, err
	}
	return newsId, nil
}

func (r *repository) UpdateUserLastViewedNews(ctx context.Context, userId string, newsId int64) error {
	const q = `UPDATE users SET last_viewed_news_id = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, q, newsId, userId)
	return err
}

func (r *repository) GetUnreadCount(ctx context.Context, userId string) (int, error) {
	const q = `
		SELECT COUNT(*)
		FROM news
		WHERE deleted_at IS NULL
		  AND status = 'published'
		  AND id > COALESCE((SELECT last_viewed_news_id FROM users WHERE id = $1), 0)`

	var count int
	if err := r.db.GetContext(ctx, &count, q, userId); err != nil {
		return 0, err
	}
	return count, nil
}
