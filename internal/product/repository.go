package product

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	CreateProduct(ctx context.Context, pc ProductCreate) (*Product, error)
	GetAll(ctx context.Context, fid string, uid string) ([]Product, error)
	GetAllByToday(ctx context.Context, fid string, uid string) ([]Product, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

func (r *repository) CreateProduct(ctx context.Context, pc ProductCreate) (*Product, error) {
	var p Product
	query := `INSERT INTO products (
	name, 
	amount,
	unit,
	calories,
	protein,
	fat,
	carbs,
	user_id,
	fit_id
	) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9
	) RETURNING *`

	err := r.db.QueryRowContext(ctx, query, pc.Name, pc.Amount, pc.Unit, pc.Calories, pc.Protein, pc.Fat, pc.Carbs, pc.UserId, pc.FitId).Scan(&p.Id, &p.Name, &p.Amount, &p.Unit, &p.Calories, &p.Protein, &p.Fat, &p.Carbs, &p.UserId, &p.FitId, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *repository) GetAll(ctx context.Context, fid string, uid string) ([]Product, error) {
	query := "SELECT * FROM products WHERE user_id = $1 AND fit_id = $2"

	rows, err := r.db.QueryContext(ctx, query, uid, fid)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ps []Product
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.Id, &p.Name, &p.Amount, &p.Unit, &p.Calories, &p.Protein, &p.Fat, &p.Carbs, &p.UserId, &p.FitId, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, nil
}

func (r *repository) GetAllByToday(ctx context.Context, fid string, uid string) ([]Product, error) {
	query := "SELECT * FROM products WHERE user_id = $1 AND fit_id = $2 AND created_at >= CURRENT_DATE AND created_at < CURRENT_DATE + INTERVAL '1 day'"

	rows, err := r.db.QueryContext(ctx, query, uid, fid)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ps []Product
	for rows.Next() {
		var p Product
		err := rows.Scan(&p.Id, &p.Name, &p.Amount, &p.Unit, &p.Calories, &p.Protein, &p.Fat, &p.Carbs, &p.UserId, &p.FitId, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, nil
}
