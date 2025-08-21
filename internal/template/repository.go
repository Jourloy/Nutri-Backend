package template

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	GetLikeName(ctx context.Context, name string) ([]Template, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

func (r *repository) GetLikeName(ctx context.Context, name string) ([]Template, error) {
	pattern := "%" + name + "%"
	query := `
	  SELECT id, name, calories, protein, fat, carbs, created_at, updated_at
	  FROM templates
	  WHERE name ILIKE $1
	  ORDER BY name
	  LIMIT 10`
	rows, err := r.db.QueryContext(ctx, query, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []Template
	for rows.Next() {
		var p Template
		if err := rows.Scan(&p.Id, &p.Name, &p.Calories, &p.Protein, &p.Fat, &p.Carbs, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		res = append(res, p)
	}
	return res, rows.Err()
}
