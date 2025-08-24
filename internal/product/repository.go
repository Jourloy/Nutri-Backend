package product

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	CreateProduct(ctx context.Context, pc ProductCreate) (*Product, error)
	GetAll(ctx context.Context, fid string, uid string) ([]Product, error)
	GetAllByToday(ctx context.Context, fid string, uid string) ([]Product, error)
	GetCount(ctx context.Context, fid string, uid string) (int, error)
	GetCountByToday(ctx context.Context, fid string, uid string) (int, error)
	GetLikeName(ctx context.Context, name string, fid string, uid string) ([]Product, error)
	UpdateProduct(ctx context.Context, pu Product, fid string, uid string) (*Product, error)
	DeleteProduct(ctx context.Context, pid int64, fid string, uid string) error
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

func (r *repository) CreateProduct(ctx context.Context, pc ProductCreate) (*Product, error) {
	const q = `
	INSERT INTO products (
		name, amount, unit, calories, protein, fat, carbs,
		basic_calories, basic_protein, basic_fat, basic_carbs,
		is_water, user_id, fit_id
	) VALUES (
		:name, :amount, :unit, :calories, :protein, :fat, :carbs,
		:basic_calories, :basic_protein, :basic_fat, :basic_carbs,
		:is_water, :user_id, :fit_id
	)
	RETURNING
		id, name, amount, unit, calories, protein, fat, carbs,
		basic_calories, basic_protein, basic_fat, basic_carbs,
		is_water, user_id, fit_id, created_at, updated_at;`

	rows, err := r.db.NamedQueryContext(ctx, q, pc)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var p Product
	if rows.Next() {
		if err := rows.StructScan(&p); err != nil {
			return nil, err
		}
		return &p, nil
	}
	return nil, errors.New("no row returned")
}

func (r *repository) GetAll(ctx context.Context, fid, uid string) ([]Product, error) {
	const q = `
	SELECT
		id, name, amount, unit, calories, protein, fat, carbs,
		basic_calories, basic_protein, basic_fat, basic_carbs,
		is_water, user_id, fit_id, created_at, updated_at
	FROM products
	WHERE user_id = $1 AND fit_id = $2
	ORDER BY created_at DESC`

	var ps []Product
	if err := r.db.SelectContext(ctx, &ps, q, uid, fid); err != nil {
		return nil, err
	}

	return ps, nil
}

func (r *repository) GetAllByToday(ctx context.Context, fid string, uid string) ([]Product, error) {
	const q = `
	SELECT
		id, name, amount, unit, calories, protein, fat, carbs,
		basic_calories, basic_protein, basic_fat, basic_carbs,
		is_water, user_id, fit_id, created_at, updated_at
	FROM products
	WHERE user_id = $1 AND fit_id = $2 AND created_at >= CURRENT_DATE AND created_at < CURRENT_DATE + INTERVAL '1 day'
	ORDER BY created_at DESC`

	var ps []Product
	if err := r.db.SelectContext(ctx, &ps, q, uid, fid); err != nil {
		return nil, err
	}

	return ps, nil
}

func (r *repository) GetCount(ctx context.Context, fid, uid string) (int, error) {
	const q = `
	SELECT COUNT(*) FROM products
	WHERE user_id = $1 AND fit_id = $2`

	var count int
	if err := r.db.GetContext(ctx, &count, q, uid, fid); err != nil {
		return 0, err
	}

	return count, nil
}

func (r *repository) GetCountByToday(ctx context.Context, fid, uid string) (int, error) {
	const q = `
	SELECT COUNT(*) FROM products
	WHERE user_id = $1 AND fit_id = $2 AND created_at >= CURRENT_DATE AND created_at < CURRENT_DATE + INTERVAL '1 day'`

	var count int
	if err := r.db.GetContext(ctx, &count, q, uid, fid); err != nil {
		return 0, err
	}

	return count, nil
}

func (r *repository) GetLikeName(ctx context.Context, name, fid, uid string) ([]Product, error) {
	pattern := "%" + name + "%"
	const q = `
	SELECT DISTINCT ON (p.name)
		p.id, p.name, p.amount, p.unit, p.calories, p.protein, p.fat, p.carbs,
		p.basic_calories, p.basic_protein, p.basic_fat, p.basic_carbs,
		p.is_water, p.user_id, p.fit_id, p.created_at, p.updated_at
	FROM products p
	WHERE p.name ILIKE $1 AND p.user_id = $2 AND p.fit_id = $3 AND basic_calories != 0
	ORDER BY p.name, p.created_at DESC
	LIMIT 10;`

	var res []Product
	if err := r.db.SelectContext(ctx, &res, q, pattern, uid, fid); err != nil {
		return nil, err
	}
	return res, nil
}

func (r *repository) UpdateProduct(ctx context.Context, pu Product, fid, uid string) (*Product, error) {
	const q = `
	UPDATE products
	SET
		name = :name,
		amount = :amount,
		unit = :unit,
		calories = :calories,
		protein = :protein,
		fat = :fat,
		carbs = :carbs,
		updated_at = now()
	WHERE id = :id AND fit_id = :fit_id AND user_id = :user_id
	RETURNING
		id, name, amount, unit, calories, protein, fat, carbs,
		basic_calories, basic_protein, basic_fat, basic_carbs,
		is_water, user_id, fit_id, created_at, updated_at;`

	args := map[string]any{
		"id": pu.Id, "fit_id": fid, "user_id": uid,
		"name": pu.Name, "amount": pu.Amount, "unit": pu.Unit,
		"calories": pu.Calories, "protein": pu.Protein, "fat": pu.Fat, "carbs": pu.Carbs,
	}

	rows, err := r.db.NamedQueryContext(ctx, q, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var p Product
		if err := rows.StructScan(&p); err != nil {
			return nil, err
		}
		return &p, nil
	}
	return nil, nil
}

func (r *repository) DeleteProduct(ctx context.Context, pid int64, fid, uid string) error {
	const q = `
	DELETE FROM products
	WHERE id = $1 AND fit_id = $2 AND user_id = $3
	RETURNING id;`

	var deletedID int64
	if err := r.db.GetContext(ctx, &deletedID, q, pid, fid, uid); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	return nil
}
