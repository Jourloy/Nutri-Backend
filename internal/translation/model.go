package translation

import "time"

type Translation struct {
	ID        int64      `db:"id" json:"id"`
	Namespace string     `db:"namespace" json:"namespace"`
	Key       string     `db:"translation_key" json:"key"`
	Locale    string     `db:"locale" json:"locale"`
	Value     string     `db:"value" json:"value"`
	CreatedAt time.Time  `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time  `db:"updated_at" json:"updatedAt"`
	DeletedAt *time.Time `db:"deleted_at" json:"deletedAt,omitempty"`
}

type UpsertRequest struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	Locale    string `json:"locale"`
	Value     string `json:"value"`
}

type DeleteRequest struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	Locale    string `json:"locale"`
}
