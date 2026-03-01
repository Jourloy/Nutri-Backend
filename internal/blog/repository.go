package blog

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jourloy/nutri-backend/internal/database"
	"github.com/lib/pq"
)

type Repository interface {
	// Categories
	CreateCategory(ctx context.Context, c CategoryCreate) (*Category, error)
	UpdateCategory(ctx context.Context, c CategoryUpdate) (*Category, error)
	DeleteCategory(ctx context.Context, id int64) error
	GetCategoryById(ctx context.Context, id int64) (*Category, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*Category, error)
	GetAllCategories(ctx context.Context) ([]Category, error)

	// Tags
	CreateTag(ctx context.Context, t TagCreate) (*Tag, error)
	UpdateTag(ctx context.Context, t TagUpdate) (*Tag, error)
	DeleteTag(ctx context.Context, id int64) error
	GetTagById(ctx context.Context, id int64) (*Tag, error)
	GetTagBySlug(ctx context.Context, slug string) (*Tag, error)
	GetAllTags(ctx context.Context) ([]Tag, error)
	GetTagsByArticleId(ctx context.Context, articleId int64) ([]Tag, error)

	// Articles
	CreateArticle(ctx context.Context, a ArticleCreate) (*Article, error)
	UpdateArticle(ctx context.Context, a ArticleUpdate) (*Article, error)
	DeleteArticle(ctx context.Context, id int64) error
	GetArticleById(ctx context.Context, id int64) (*Article, error)
	GetArticleBySlug(ctx context.Context, slug string) (*Article, error)
	GetArticles(ctx context.Context, params ArticleListParams, includeAll bool) (*ArticleListResponse, error)
	IncrementViewCount(ctx context.Context, id int64) error
	UpdateReadingTime(ctx context.Context, id int64, minutes int) error
	SetPublishedAt(ctx context.Context, id int64) error

	// Article Tags
	SetArticleTags(ctx context.Context, articleId int64, tagIds []int64) error

	// Feedback
	CreateFeedback(ctx context.Context, f FeedbackCreate) (*Feedback, error)
	GetFeedbackByUser(ctx context.Context, articleId int64, userId string) (*Feedback, error)
	GetFeedbackBySession(ctx context.Context, articleId int64, sessionId string) (*Feedback, error)
	GetFeedbackStats(ctx context.Context, articleId int64) (*FeedbackStats, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

// ===== Categories =====

func (r *repository) CreateCategory(ctx context.Context, c CategoryCreate) (*Category, error) {
	const q = `
		INSERT INTO blog_categories (slug, name_ru, name_en, description_ru, description_en, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, slug, name_ru, name_en, description_ru, description_en, sort_order, created_at, updated_at`

	var cat Category
	err := r.db.GetContext(ctx, &cat, q, c.Slug, c.NameRu, c.NameEn, c.DescriptionRu, c.DescriptionEn, c.SortOrder)
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *repository) UpdateCategory(ctx context.Context, c CategoryUpdate) (*Category, error) {
	const q = `
		UPDATE blog_categories SET
			slug = $1,
			name_ru = $2,
			name_en = $3,
			description_ru = $4,
			description_en = $5,
			sort_order = $6,
			updated_at = NOW()
		WHERE id = $7 AND deleted_at IS NULL
		RETURNING id, slug, name_ru, name_en, description_ru, description_en, sort_order, created_at, updated_at`

	var cat Category
	err := r.db.GetContext(ctx, &cat, q, c.Slug, c.NameRu, c.NameEn, c.DescriptionRu, c.DescriptionEn, c.SortOrder, c.Id)
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *repository) DeleteCategory(ctx context.Context, id int64) error {
	const q = `UPDATE blog_categories SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *repository) GetCategoryById(ctx context.Context, id int64) (*Category, error) {
	const q = `
		SELECT id, slug, name_ru, name_en, description_ru, description_en, sort_order, created_at, updated_at
		FROM blog_categories
		WHERE id = $1 AND deleted_at IS NULL`

	var cat Category
	if err := r.db.GetContext(ctx, &cat, q, id); err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *repository) GetCategoryBySlug(ctx context.Context, slug string) (*Category, error) {
	const q = `
		SELECT id, slug, name_ru, name_en, description_ru, description_en, sort_order, created_at, updated_at
		FROM blog_categories
		WHERE slug = $1 AND deleted_at IS NULL`

	var cat Category
	if err := r.db.GetContext(ctx, &cat, q, slug); err != nil {
		return nil, err
	}
	return &cat, nil
}

func (r *repository) GetAllCategories(ctx context.Context) ([]Category, error) {
	const q = `
		SELECT id, slug, name_ru, name_en, description_ru, description_en, sort_order, created_at, updated_at
		FROM blog_categories
		WHERE deleted_at IS NULL
		ORDER BY sort_order ASC, name_ru ASC`

	var cats []Category
	if err := r.db.SelectContext(ctx, &cats, q); err != nil {
		return nil, err
	}
	return cats, nil
}

// ===== Tags =====

func (r *repository) CreateTag(ctx context.Context, t TagCreate) (*Tag, error) {
	const q = `
		INSERT INTO blog_tags (slug, name_ru, name_en)
		VALUES ($1, $2, $3)
		RETURNING id, slug, name_ru, name_en, created_at, updated_at`

	var tag Tag
	err := r.db.GetContext(ctx, &tag, q, t.Slug, t.NameRu, t.NameEn)
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *repository) UpdateTag(ctx context.Context, t TagUpdate) (*Tag, error) {
	const q = `
		UPDATE blog_tags SET
			slug = $1,
			name_ru = $2,
			name_en = $3,
			updated_at = NOW()
		WHERE id = $4 AND deleted_at IS NULL
		RETURNING id, slug, name_ru, name_en, created_at, updated_at`

	var tag Tag
	err := r.db.GetContext(ctx, &tag, q, t.Slug, t.NameRu, t.NameEn, t.Id)
	if err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *repository) DeleteTag(ctx context.Context, id int64) error {
	const q = `UPDATE blog_tags SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *repository) GetTagById(ctx context.Context, id int64) (*Tag, error) {
	const q = `
		SELECT id, slug, name_ru, name_en, created_at, updated_at
		FROM blog_tags
		WHERE id = $1 AND deleted_at IS NULL`

	var tag Tag
	if err := r.db.GetContext(ctx, &tag, q, id); err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *repository) GetTagBySlug(ctx context.Context, slug string) (*Tag, error) {
	const q = `
		SELECT id, slug, name_ru, name_en, created_at, updated_at
		FROM blog_tags
		WHERE slug = $1 AND deleted_at IS NULL`

	var tag Tag
	if err := r.db.GetContext(ctx, &tag, q, slug); err != nil {
		return nil, err
	}
	return &tag, nil
}

func (r *repository) GetAllTags(ctx context.Context) ([]Tag, error) {
	const q = `
		SELECT id, slug, name_ru, name_en, created_at, updated_at
		FROM blog_tags
		WHERE deleted_at IS NULL
		ORDER BY name_ru ASC`

	var tags []Tag
	if err := r.db.SelectContext(ctx, &tags, q); err != nil {
		return nil, err
	}
	return tags, nil
}

func (r *repository) GetTagsByArticleId(ctx context.Context, articleId int64) ([]Tag, error) {
	const q = `
		SELECT t.id, t.slug, t.name_ru, t.name_en, t.created_at, t.updated_at
		FROM blog_tags t
		JOIN blog_article_tags at ON at.tag_id = t.id
		WHERE at.article_id = $1 AND t.deleted_at IS NULL
		ORDER BY t.name_ru ASC`

	var tags []Tag
	if err := r.db.SelectContext(ctx, &tags, q, articleId); err != nil {
		return nil, err
	}
	return tags, nil
}

// ===== Articles =====

func (r *repository) CreateArticle(ctx context.Context, a ArticleCreate) (*Article, error) {
	const q = `
		INSERT INTO blog_articles (
			slug, title_ru, title_en, content_ru, content_en,
			preview_text_ru, preview_text_en, preview_image_url,
			category_id, meta_description_ru, meta_description_en,
			og_image_url, canonical_url, sources, status, author_id, reading_time_minutes
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
		RETURNING id, slug, title_ru, title_en, content_ru, content_en,
		          preview_text_ru, preview_text_en, preview_image_url,
		          category_id, meta_description_ru, meta_description_en,
		          og_image_url, canonical_url, sources, status, view_count, reading_time_minutes,
		          published_at, author_id, created_at, updated_at`

	readingTime := CalculateReadingTime(a.ContentRu, a.ContentEn)

	var article Article
	err := r.db.GetContext(ctx, &article, q,
		a.Slug, a.TitleRu, a.TitleEn, a.ContentRu, a.ContentEn,
		a.PreviewTextRu, a.PreviewTextEn, a.PreviewImageUrl,
		a.CategoryId, a.MetaDescriptionRu, a.MetaDescriptionEn,
		a.OgImageUrl, a.CanonicalUrl, pq.Array(a.Sources), a.Status, a.AuthorId, readingTime)
	if err != nil {
		return nil, err
	}
	return &article, nil
}

func (r *repository) UpdateArticle(ctx context.Context, a ArticleUpdate) (*Article, error) {
	const q = `
		UPDATE blog_articles SET
			slug = $1,
			title_ru = $2,
			title_en = $3,
			content_ru = $4,
			content_en = $5,
			preview_text_ru = $6,
			preview_text_en = $7,
			preview_image_url = $8,
			category_id = $9,
			meta_description_ru = $10,
			meta_description_en = $11,
			og_image_url = $12,
			canonical_url = $13,
			sources = $14,
			status = $15,
			reading_time_minutes = $16,
			updated_at = NOW()
		WHERE id = $17 AND deleted_at IS NULL
		RETURNING id, slug, title_ru, title_en, content_ru, content_en,
		          preview_text_ru, preview_text_en, preview_image_url,
		          category_id, meta_description_ru, meta_description_en,
		          og_image_url, canonical_url, sources, status, view_count, reading_time_minutes,
		          published_at, author_id, created_at, updated_at`

	readingTime := CalculateReadingTime(a.ContentRu, a.ContentEn)

	var article Article
	err := r.db.GetContext(ctx, &article, q,
		a.Slug, a.TitleRu, a.TitleEn, a.ContentRu, a.ContentEn,
		a.PreviewTextRu, a.PreviewTextEn, a.PreviewImageUrl,
		a.CategoryId, a.MetaDescriptionRu, a.MetaDescriptionEn,
		a.OgImageUrl, a.CanonicalUrl, pq.Array(a.Sources), a.Status, readingTime, a.Id)
	if err != nil {
		return nil, err
	}
	return &article, nil
}

func (r *repository) DeleteArticle(ctx context.Context, id int64) error {
	const q = `UPDATE blog_articles SET deleted_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *repository) GetArticleById(ctx context.Context, id int64) (*Article, error) {
	const q = `
		SELECT id, slug, title_ru, title_en, content_ru, content_en,
		       preview_text_ru, preview_text_en, preview_image_url,
		       category_id, meta_description_ru, meta_description_en,
		       og_image_url, canonical_url, sources, status, view_count, reading_time_minutes,
		       published_at, author_id, created_at, updated_at
		FROM blog_articles
		WHERE id = $1 AND deleted_at IS NULL`

	var article Article
	if err := r.db.GetContext(ctx, &article, q, id); err != nil {
		return nil, err
	}
	return &article, nil
}

func (r *repository) GetArticleBySlug(ctx context.Context, slug string) (*Article, error) {
	const q = `
		SELECT id, slug, title_ru, title_en, content_ru, content_en,
		       preview_text_ru, preview_text_en, preview_image_url,
		       category_id, meta_description_ru, meta_description_en,
		       og_image_url, canonical_url, sources, status, view_count, reading_time_minutes,
		       published_at, author_id, created_at, updated_at
		FROM blog_articles
		WHERE slug = $1 AND deleted_at IS NULL`

	var article Article
	if err := r.db.GetContext(ctx, &article, q, slug); err != nil {
		return nil, err
	}
	return &article, nil
}

func (r *repository) GetArticles(ctx context.Context, params ArticleListParams, includeAll bool) (*ArticleListResponse, error) {
	// Build WHERE conditions
	conditions := []string{"a.deleted_at IS NULL"}
	args := []interface{}{}
	argIndex := 1

	if len(params.AllowedStatuses) > 0 {
		conditions = append(conditions, fmt.Sprintf("a.status = ANY($%d)", argIndex))
		args = append(args, pq.Array(params.AllowedStatuses))
		argIndex++
	} else if !includeAll {
		conditions = append(conditions, "a.status != 'draft'")
	}

	if params.Status != nil && *params.Status != "" {
		conditions = append(conditions, fmt.Sprintf("a.status = $%d", argIndex))
		args = append(args, *params.Status)
		argIndex++
	}

	if params.CategorySlug != nil && *params.CategorySlug != "" {
		conditions = append(conditions, fmt.Sprintf("c.slug = $%d", argIndex))
		args = append(args, *params.CategorySlug)
		argIndex++
	}

	if params.TagSlug != nil && *params.TagSlug != "" {
		conditions = append(conditions, fmt.Sprintf(`
			EXISTS (
				SELECT 1 FROM blog_article_tags at2
				JOIN blog_tags t2 ON t2.id = at2.tag_id
				WHERE at2.article_id = a.id AND t2.slug = $%d
			)`, argIndex))
		args = append(args, *params.TagSlug)
		argIndex++
	}

	if params.Search != nil && *params.Search != "" {
		searchPattern := "%" + strings.ToLower(*params.Search) + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(LOWER(a.title_ru) LIKE $%d OR LOWER(a.title_en) LIKE $%d OR LOWER(a.content_ru) LIKE $%d OR LOWER(a.content_en) LIKE $%d)",
			argIndex, argIndex, argIndex, argIndex))
		args = append(args, searchPattern)
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM blog_articles a
		LEFT JOIN blog_categories c ON c.id = a.category_id
		WHERE %s`, whereClause)

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, err
	}

	// Pagination
	page := params.Page
	if page < 1 {
		page = 1
	}
	perPage := params.PerPage
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage

	// Fetch articles
	selectQuery := fmt.Sprintf(`
		SELECT a.id, a.slug, a.title_ru, a.title_en, a.content_ru, a.content_en,
		       a.preview_text_ru, a.preview_text_en, a.preview_image_url,
		       a.category_id, a.meta_description_ru, a.meta_description_en,
		       a.og_image_url, a.canonical_url, a.sources, a.status, a.view_count, a.reading_time_minutes,
		       a.published_at, a.author_id, a.created_at, a.updated_at
		FROM blog_articles a
		LEFT JOIN blog_categories c ON c.id = a.category_id
		WHERE %s
		ORDER BY a.published_at DESC NULLS LAST, a.created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, perPage, offset)

	var articles []Article
	if err := r.db.SelectContext(ctx, &articles, selectQuery, args...); err != nil {
		return nil, err
	}

	totalPages := (total + perPage - 1) / perPage

	return &ArticleListResponse{
		Articles:   articles,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

func (r *repository) IncrementViewCount(ctx context.Context, id int64) error {
	const q = `UPDATE blog_articles SET view_count = view_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *repository) UpdateReadingTime(ctx context.Context, id int64, minutes int) error {
	const q = `UPDATE blog_articles SET reading_time_minutes = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, q, minutes, id)
	return err
}

func (r *repository) SetPublishedAt(ctx context.Context, id int64) error {
	const q = `UPDATE blog_articles SET published_at = NOW() WHERE id = $1 AND published_at IS NULL`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// ===== Article Tags =====

func (r *repository) SetArticleTags(ctx context.Context, articleId int64, tagIds []int64) error {
	// Delete existing tags
	_, err := r.db.ExecContext(ctx, `DELETE FROM blog_article_tags WHERE article_id = $1`, articleId)
	if err != nil {
		return err
	}

	// Insert new tags
	if len(tagIds) == 0 {
		return nil
	}

	query := `INSERT INTO blog_article_tags (article_id, tag_id) VALUES `
	values := []string{}
	args := []interface{}{}
	for i, tagId := range tagIds {
		values = append(values, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		args = append(args, articleId, tagId)
	}
	query += strings.Join(values, ", ") + " ON CONFLICT DO NOTHING"

	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

// ===== Feedback =====

func (r *repository) CreateFeedback(ctx context.Context, f FeedbackCreate) (*Feedback, error) {
	const q = `
		INSERT INTO blog_article_feedback (article_id, user_id, session_id, helpful)
		VALUES ($1, $2, $3, $4)
		RETURNING id, article_id, user_id, session_id, helpful, created_at`

	var feedback Feedback
	err := r.db.GetContext(ctx, &feedback, q, f.ArticleId, f.UserId, f.SessionId, f.Helpful)
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}

func (r *repository) GetFeedbackByUser(ctx context.Context, articleId int64, userId string) (*Feedback, error) {
	const q = `
		SELECT id, article_id, user_id, session_id, helpful, created_at
		FROM blog_article_feedback
		WHERE article_id = $1 AND user_id = $2`

	var feedback Feedback
	if err := r.db.GetContext(ctx, &feedback, q, articleId, userId); err != nil {
		return nil, err
	}
	return &feedback, nil
}

func (r *repository) GetFeedbackBySession(ctx context.Context, articleId int64, sessionId string) (*Feedback, error) {
	const q = `
		SELECT id, article_id, user_id, session_id, helpful, created_at
		FROM blog_article_feedback
		WHERE article_id = $1 AND session_id = $2 AND user_id IS NULL`

	var feedback Feedback
	if err := r.db.GetContext(ctx, &feedback, q, articleId, sessionId); err != nil {
		return nil, err
	}
	return &feedback, nil
}

func (r *repository) GetFeedbackStats(ctx context.Context, articleId int64) (*FeedbackStats, error) {
	const q = `
		SELECT
			COUNT(*) FILTER (WHERE helpful = true) as helpful_count,
			COUNT(*) FILTER (WHERE helpful = false) as not_helpful_count,
			COUNT(*) as total_count
		FROM blog_article_feedback
		WHERE article_id = $1`

	type statsRow struct {
		HelpfulCount    int `db:"helpful_count"`
		NotHelpfulCount int `db:"not_helpful_count"`
		TotalCount      int `db:"total_count"`
	}

	var row statsRow
	if err := r.db.GetContext(ctx, &row, q, articleId); err != nil {
		return nil, err
	}

	return &FeedbackStats{
		ArticleId:       articleId,
		HelpfulCount:    row.HelpfulCount,
		NotHelpfulCount: row.NotHelpfulCount,
		TotalCount:      row.TotalCount,
	}, nil
}

// ===== Helpers =====

// CalculateReadingTime calculates reading time based on word count.
// Average reading speed: 200 words/minute.
func CalculateReadingTime(contentRu, contentEn string) int {
	// Count words in both languages, take the max
	wordsRu := len(strings.Fields(contentRu))
	wordsEn := len(strings.Fields(contentEn))
	words := wordsRu
	if wordsEn > words {
		words = wordsEn
	}
	minutes := words / 200
	if minutes < 1 {
		minutes = 1
	}
	return minutes
}
