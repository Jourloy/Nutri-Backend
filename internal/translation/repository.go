package translation

import (
	"context"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return Repository{db: database.Database}
}

func (r Repository) GetByLocale(ctx context.Context, locale string) ([]Translation, error) {
	var translations []Translation
	query := `
		SELECT id, namespace, translation_key, locale, value, created_at, updated_at, deleted_at
		FROM translations
		WHERE locale = $1 AND deleted_at IS NULL
	`
	err := r.db.SelectContext(ctx, &translations, query, locale)
	if err != nil {
		return nil, err
	}
	return translations, nil
}

func (r Repository) Upsert(ctx context.Context, req UpsertRequest) (*Translation, error) {
	query := `
		INSERT INTO translations (namespace, translation_key, locale, value)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (namespace, translation_key, locale)
		DO UPDATE SET value = EXCLUDED.value, updated_at = NOW(), deleted_at = NULL
		RETURNING id, namespace, translation_key, locale, value, created_at, updated_at, deleted_at
	`
	var t Translation
	if err := r.db.GetContext(ctx, &t, query, req.Namespace, req.Key, strings.ToLower(req.Locale), req.Value); err != nil {
		return nil, err
	}
	return &t, nil
}

func (r Repository) SoftDelete(ctx context.Context, req DeleteRequest) error {
	query := `
		UPDATE translations
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE namespace = $1 AND translation_key = $2 AND locale = $3 AND deleted_at IS NULL
	`
	result, err := r.db.ExecContext(ctx, query, req.Namespace, req.Key, strings.ToLower(req.Locale))
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("translation not found")
	}
	return nil
}
