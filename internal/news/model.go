package news

import "time"

// News represents a news or update item
type News struct {
	Id          int64      `json:"id" db:"id"`
	TitleRu     string     `json:"titleRu" db:"title_ru"`
	TitleEn     string     `json:"titleEn" db:"title_en"`
	ContentRu   string     `json:"contentRu" db:"content_ru"` // HTML from TipTap
	ContentEn   string     `json:"contentEn" db:"content_en"` // HTML from TipTap
	ImageUrl    *string    `json:"imageUrl,omitempty" db:"image_url"`
	Status      string     `json:"status" db:"status"` // draft, preview, published
	IsPublished bool       `json:"isPublished" db:"is_published"`
	PublishedAt *time.Time `json:"publishedAt,omitempty" db:"published_at"`
	Priority    int        `json:"priority" db:"priority"`
	CreatedAt   time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
}

// NewsCreate represents data for creating news
type NewsCreate struct {
	TitleRu   string  `json:"titleRu" db:"title_ru"`
	TitleEn   string  `json:"titleEn" db:"title_en"`
	ContentRu string  `json:"contentRu" db:"content_ru"`
	ContentEn string  `json:"contentEn" db:"content_en"`
	ImageUrl  *string `json:"imageUrl,omitempty" db:"image_url"`
	Priority  int     `json:"priority" db:"priority"`
}

// NewsUpdate represents data for updating news
type NewsUpdate struct {
	Id        int64   `json:"id" db:"id"`
	TitleRu   string  `json:"titleRu" db:"title_ru"`
	TitleEn   string  `json:"titleEn" db:"title_en"`
	ContentRu string  `json:"contentRu" db:"content_ru"`
	ContentEn string  `json:"contentEn" db:"content_en"`
	ImageUrl  *string `json:"imageUrl,omitempty" db:"image_url"`
	Priority  int     `json:"priority" db:"priority"`
}

// NewsListResponse represents response with news list and unread count
type NewsListResponse struct {
	News         []News `json:"news"`
	UnreadCount  int    `json:"unreadCount"`
	LastViewedId *int64 `json:"lastViewedId,omitempty"`
}
