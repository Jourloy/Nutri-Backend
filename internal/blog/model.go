package blog

import (
	"time"

	"github.com/lib/pq"
)

// ===== Category =====

type Category struct {
	Id            int64      `json:"id" db:"id"`
	Slug          string     `json:"slug" db:"slug"`
	NameRu        string     `json:"nameRu" db:"name_ru"`
	NameEn        string     `json:"nameEn" db:"name_en"`
	DescriptionRu *string    `json:"descriptionRu,omitempty" db:"description_ru"`
	DescriptionEn *string    `json:"descriptionEn,omitempty" db:"description_en"`
	SortOrder     int        `json:"sortOrder" db:"sort_order"`
	CreatedAt     time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt     *time.Time `json:"-" db:"deleted_at"`
}

type CategoryCreate struct {
	Slug          string  `json:"slug"`
	NameRu        string  `json:"nameRu"`
	NameEn        string  `json:"nameEn"`
	DescriptionRu *string `json:"descriptionRu,omitempty"`
	DescriptionEn *string `json:"descriptionEn,omitempty"`
	SortOrder     int     `json:"sortOrder"`
}

type CategoryUpdate struct {
	Id            int64   `json:"id"`
	Slug          string  `json:"slug"`
	NameRu        string  `json:"nameRu"`
	NameEn        string  `json:"nameEn"`
	DescriptionRu *string `json:"descriptionRu,omitempty"`
	DescriptionEn *string `json:"descriptionEn,omitempty"`
	SortOrder     int     `json:"sortOrder"`
}

// ===== Tag =====

type Tag struct {
	Id        int64      `json:"id" db:"id"`
	Slug      string     `json:"slug" db:"slug"`
	NameRu    string     `json:"nameRu" db:"name_ru"`
	NameEn    string     `json:"nameEn" db:"name_en"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt *time.Time `json:"-" db:"deleted_at"`
}

type TagCreate struct {
	Slug   string `json:"slug"`
	NameRu string `json:"nameRu"`
	NameEn string `json:"nameEn"`
}

type TagUpdate struct {
	Id     int64  `json:"id"`
	Slug   string `json:"slug"`
	NameRu string `json:"nameRu"`
	NameEn string `json:"nameEn"`
}

// ===== Article =====

type Article struct {
	Id                 int64          `json:"id" db:"id"`
	Slug               string         `json:"slug" db:"slug"`
	TitleRu            string         `json:"titleRu" db:"title_ru"`
	TitleEn            string         `json:"titleEn" db:"title_en"`
	ContentRu          string         `json:"contentRu" db:"content_ru"`
	ContentEn          string         `json:"contentEn" db:"content_en"`
	PreviewTextRu      *string        `json:"previewTextRu,omitempty" db:"preview_text_ru"`
	PreviewTextEn      *string        `json:"previewTextEn,omitempty" db:"preview_text_en"`
	PreviewImageUrl    *string        `json:"previewImageUrl,omitempty" db:"preview_image_url"`
	CategoryId         *int64         `json:"categoryId,omitempty" db:"category_id"`
	MetaDescriptionRu  *string        `json:"metaDescriptionRu,omitempty" db:"meta_description_ru"`
	MetaDescriptionEn  *string        `json:"metaDescriptionEn,omitempty" db:"meta_description_en"`
	OgImageUrl         *string        `json:"ogImageUrl,omitempty" db:"og_image_url"`
	CanonicalUrl       *string        `json:"canonicalUrl,omitempty" db:"canonical_url"`
	Sources            pq.StringArray `json:"sources,omitempty" db:"sources"`
	Status             string         `json:"status" db:"status"`
	ViewCount          int            `json:"viewCount" db:"view_count"`
	ReadingTimeMinutes int            `json:"readingTimeMinutes" db:"reading_time_minutes"`
	PublishedAt        *time.Time     `json:"publishedAt,omitempty" db:"published_at"`
	AuthorId           *string        `json:"authorId,omitempty" db:"author_id"`
	CreatedAt          time.Time      `json:"createdAt" db:"created_at"`
	UpdatedAt          time.Time      `json:"updatedAt" db:"updated_at"`
	DeletedAt          *time.Time     `json:"-" db:"deleted_at"`

	// Joined fields (not in DB directly)
	Category *Category `json:"category,omitempty" db:"-"`
	Tags     []Tag     `json:"tags,omitempty" db:"-"`
}

// ArticlePublic is the public view of an article (without viewCount for non-admins)
type ArticlePublic struct {
	Id                 int64      `json:"id"`
	Slug               string     `json:"slug"`
	TitleRu            string     `json:"titleRu"`
	TitleEn            string     `json:"titleEn"`
	ContentRu          string     `json:"contentRu"`
	ContentEn          string     `json:"contentEn"`
	PreviewTextRu      *string    `json:"previewTextRu,omitempty"`
	PreviewTextEn      *string    `json:"previewTextEn,omitempty"`
	PreviewImageUrl    *string    `json:"previewImageUrl,omitempty"`
	CategoryId         *int64     `json:"categoryId,omitempty"`
	MetaDescriptionRu  *string    `json:"metaDescriptionRu,omitempty"`
	MetaDescriptionEn  *string    `json:"metaDescriptionEn,omitempty"`
	OgImageUrl         *string    `json:"ogImageUrl,omitempty"`
	CanonicalUrl       *string    `json:"canonicalUrl,omitempty"`
	Status             string     `json:"status"`
	ReadingTimeMinutes int        `json:"readingTimeMinutes"`
	PublishedAt        *time.Time `json:"publishedAt,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
	Category           *Category  `json:"category,omitempty"`
	Tags               []Tag      `json:"tags,omitempty"`
}

func (a *Article) ToPublic() ArticlePublic {
	return ArticlePublic{
		Id:                 a.Id,
		Slug:               a.Slug,
		TitleRu:            a.TitleRu,
		TitleEn:            a.TitleEn,
		ContentRu:          a.ContentRu,
		ContentEn:          a.ContentEn,
		PreviewTextRu:      a.PreviewTextRu,
		PreviewTextEn:      a.PreviewTextEn,
		PreviewImageUrl:    a.PreviewImageUrl,
		CategoryId:         a.CategoryId,
		MetaDescriptionRu:  a.MetaDescriptionRu,
		MetaDescriptionEn:  a.MetaDescriptionEn,
		OgImageUrl:         a.OgImageUrl,
		CanonicalUrl:       a.CanonicalUrl,
		Status:             a.Status,
		ReadingTimeMinutes: a.ReadingTimeMinutes,
		PublishedAt:        a.PublishedAt,
		CreatedAt:          a.CreatedAt,
		Category:           a.Category,
		Tags:               a.Tags,
	}
}

type ArticleCreate struct {
	Slug              string   `json:"slug"`
	TitleRu           string   `json:"titleRu"`
	TitleEn           string   `json:"titleEn"`
	ContentRu         string   `json:"contentRu"`
	ContentEn         string   `json:"contentEn"`
	PreviewTextRu     *string  `json:"previewTextRu,omitempty"`
	PreviewTextEn     *string  `json:"previewTextEn,omitempty"`
	PreviewImageUrl   *string  `json:"previewImageUrl,omitempty"`
	CategoryId        *int64   `json:"categoryId,omitempty"`
	MetaDescriptionRu *string  `json:"metaDescriptionRu,omitempty"`
	MetaDescriptionEn *string  `json:"metaDescriptionEn,omitempty"`
	OgImageUrl        *string  `json:"ogImageUrl,omitempty"`
	CanonicalUrl      *string  `json:"canonicalUrl,omitempty"`
	Sources           []string `json:"sources,omitempty"`
	Status            string   `json:"status"`
	AuthorId          *string  `json:"-"`
	TagIds            []int64  `json:"tagIds,omitempty"`
}

type ArticleUpdate struct {
	Id                int64    `json:"id"`
	Slug              string   `json:"slug"`
	TitleRu           string   `json:"titleRu"`
	TitleEn           string   `json:"titleEn"`
	ContentRu         string   `json:"contentRu"`
	ContentEn         string   `json:"contentEn"`
	PreviewTextRu     *string  `json:"previewTextRu,omitempty"`
	PreviewTextEn     *string  `json:"previewTextEn,omitempty"`
	PreviewImageUrl   *string  `json:"previewImageUrl,omitempty"`
	CategoryId        *int64   `json:"categoryId,omitempty"`
	MetaDescriptionRu *string  `json:"metaDescriptionRu,omitempty"`
	MetaDescriptionEn *string  `json:"metaDescriptionEn,omitempty"`
	OgImageUrl        *string  `json:"ogImageUrl,omitempty"`
	CanonicalUrl      *string  `json:"canonicalUrl,omitempty"`
	Sources           []string `json:"sources,omitempty"`
	Status            string   `json:"status"`
	TagIds            []int64  `json:"tagIds,omitempty"`
}

// ===== Feedback =====

type Feedback struct {
	Id        string    `json:"id" db:"id"`
	ArticleId int64     `json:"articleId" db:"article_id"`
	UserId    *string   `json:"userId,omitempty" db:"user_id"`
	SessionId *string   `json:"sessionId,omitempty" db:"session_id"`
	Helpful   bool      `json:"helpful" db:"helpful"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

type FeedbackCreate struct {
	ArticleId int64   `json:"articleId"`
	UserId    *string `json:"-"`
	SessionId *string `json:"sessionId,omitempty"`
	Helpful   bool    `json:"helpful"`
}

type FeedbackStats struct {
	ArticleId       int64 `json:"articleId"`
	HelpfulCount    int   `json:"helpfulCount"`
	NotHelpfulCount int   `json:"notHelpfulCount"`
	TotalCount      int   `json:"totalCount"`
}

// ===== List Responses =====

type ArticleListResponse struct {
	Articles   []Article `json:"articles"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PerPage    int       `json:"perPage"`
	TotalPages int       `json:"totalPages"`
}

type ArticlePublicListResponse struct {
	Articles   []ArticlePublic `json:"articles"`
	Total      int             `json:"total"`
	Page       int             `json:"page"`
	PerPage    int             `json:"perPage"`
	TotalPages int             `json:"totalPages"`
}

type ViewerAccess struct {
	IsAdmin         bool
	IsAuthenticated bool
	PlanCode        string
}

type ArticleListParams struct {
	Page            int
	PerPage         int
	CategorySlug    *string
	TagSlug         *string
	Status          *string
	Search          *string
	AllowedStatuses []string
}

// ===== Image Upload =====

type ImageUploadResponse struct {
	Url string `json:"url"`
}
