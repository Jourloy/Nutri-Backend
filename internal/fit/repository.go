package fit

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	CreateFitProfile(ctx context.Context, fc FitProfileCreate) (*FitProfile, error)
	GetFitProfileByUser(ctx context.Context, uid string) (*FitProfile, error)
	GetFitProfileById(ctx context.Context, id string) (*FitProfile, error)
	UpdateFitProfile(ctx context.Context, fu FitProfileCreate, uid string, fid string) (*FitProfile, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

// единый список колонок — без SELECT *
const fitColumns = `
	id, age, gender, height, weight, activity_level, goal,
	calories, protein, fat, carbs, water_limit,
	user_id, created_at, updated_at, deleted_at
`

func (r *repository) CreateFitProfile(ctx context.Context, fc FitProfileCreate) (*FitProfile, error) {
	const q = `
	INSERT INTO fit_profiles (
		age, gender, height, weight, activity_level, goal,
		calories, protein, fat, carbs, water_limit, user_id
	) VALUES (
		:age, :gender, :height, :weight, :activity_level, :goal,
		:calories, :protein, :fat, :carbs, :water_limit, :user_id
	)
	RETURNING ` + fitColumns + `;`

	args := map[string]any{
		"age":            fc.Age,
		"gender":         fc.Gender,
		"height":         fc.Height,
		"weight":         fc.Weight,
		"activity_level": fc.ActivityLevel,
		"goal":           fc.Goal,
		"calories":       fc.Calories,
		"protein":        fc.Protein,
		"fat":            fc.Fat,
		"carbs":          fc.Carbs,
		"water_limit":    fc.WaterLimit,
		"user_id":        fc.UserId,
	}

	rows, err := r.db.NamedQueryContext(ctx, q, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var f FitProfile
		if err := rows.StructScan(&f); err != nil {
			return nil, err
		}
		return &f, nil
	}
	return nil, nil
}

func (r *repository) UpdateFitProfile(ctx context.Context, fu FitProfileCreate, uid, fid string) (*FitProfile, error) {
	const q = `
	UPDATE fit_profiles
	SET
		age = :age,
		gender = :gender,
		height = :height,
		weight = :weight,
		activity_level = :activity_level,
		goal = :goal,
		calories = :calories,
		protein = :protein,
		fat = :fat,
		carbs = :carbs,
		water_limit = :water_limit,
		updated_at = now()
	WHERE id = :id AND user_id = :user_id
	RETURNING ` + fitColumns + `;`

	args := map[string]any{
		"id":             fid,
		"user_id":        uid,
		"age":            fu.Age,
		"gender":         fu.Gender,
		"height":         fu.Height,
		"weight":         fu.Weight,
		"activity_level": fu.ActivityLevel,
		"goal":           fu.Goal,
		"calories":       fu.Calories,
		"protein":        fu.Protein,
		"fat":            fu.Fat,
		"carbs":          fu.Carbs,
		"water_limit":    fu.WaterLimit,
	}

	rows, err := r.db.NamedQueryContext(ctx, q, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var f FitProfile
		if err := rows.StructScan(&f); err != nil {
			return nil, err
		}
		return &f, nil
	}
	// not found
	return nil, nil
}

func (r *repository) GetFitProfileByUser(ctx context.Context, uid string) (*FitProfile, error) {
	const q = `SELECT ` + fitColumns + ` FROM fit_profiles WHERE user_id = $1 LIMIT 1;`

	var f FitProfile
	if err := r.db.GetContext(ctx, &f, q, uid); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *repository) GetFitProfileById(ctx context.Context, id string) (*FitProfile, error) {
	const q = `SELECT ` + fitColumns + ` FROM fit_profiles WHERE id = $1 LIMIT 1;`

	var f FitProfile
	if err := r.db.GetContext(ctx, &f, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}
