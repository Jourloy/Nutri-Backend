package feature

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	Create(ctx context.Context, f Feature) (*Feature, error)
	Update(ctx context.Context, f Feature) (*Feature, error)
	Delete(ctx context.Context, key string) error
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

func (r *repository) Create(ctx context.Context, f Feature) (*Feature, error) {
	const q = `
        INSERT INTO features (key, name, description, unit)
        VALUES (:key, :name, :description, :unit)
        RETURNING key, name, description, unit;`

	rows, err := r.db.NamedQueryContext(ctx, q, f)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var res Feature
		if err := rows.StructScan(&res); err != nil {
			return nil, err
		}
		return &res, nil
	}
	return nil, nil
}

func (r *repository) Update(ctx context.Context, f Feature) (*Feature, error) {
	const q = `
        UPDATE features
        SET name = :name, description = :description, unit = :unit
        WHERE key = :key
        RETURNING key, name, description, unit;`

	rows, err := r.db.NamedQueryContext(ctx, q, f)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var res Feature
		if err := rows.StructScan(&res); err != nil {
			return nil, err
		}
		return &res, nil
	}
	return nil, nil
}

func (r *repository) Delete(ctx context.Context, key string) error {
	const q = `DELETE FROM features WHERE key = $1 RETURNING key;`

	var deletedKey string
	if err := r.db.GetContext(ctx, &deletedKey, q, key); err != nil {
		return err
	}
	return nil
}
