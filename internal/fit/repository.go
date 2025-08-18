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
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

func (r *repository) CreateFitProfile(ctx context.Context, fc FitProfileCreate) (*FitProfile, error) {

	var f FitProfile
	query := `INSERT INTO fit_profiles (
	age, 
	gender,
	height,
	weight,
	activity_level,
	goal,
	calories,
	protein,
	fat,
	carbs,
	user_id
	) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
	) RETURNING *`

	err := r.db.QueryRowContext(ctx, query, fc.Age, fc.Gender, fc.Height, fc.Weight, fc.ActivityLevel, fc.Goal, fc.Calories, fc.Protein, fc.Fat, fc.Carbs, fc.UserId).Scan(&f.Id, &f.Age, &f.Gender, &f.Height, &f.Weight, &f.ActivityLevel, &f.Goal, &f.Calories, &f.Protein, &f.Fat, &f.Carbs, &f.UserId, &f.CreatedAt, &f.UpdatedAt, &f.DeletedAt)
	if err != nil {
		return nil, err
	}

	return &f, nil
}

func (r *repository) GetFitProfileByUser(ctx context.Context, uid string) (*FitProfile, error) {
	query := "SELECT * FROM fit_profiles WHERE user_id = $1"
	row := r.db.QueryRowContext(ctx, query, uid)

	var f FitProfile
	err := row.Scan(&f.Id, &f.Age, &f.Gender, &f.Height, &f.Weight, &f.ActivityLevel, &f.Goal, &f.Calories, &f.Protein, &f.Fat, &f.Carbs, &f.UserId, &f.CreatedAt, &f.UpdatedAt, &f.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *repository) GetFitProfileById(ctx context.Context, id string) (*FitProfile, error) {
	query := "SELECT * FROM fit_profiles WHERE id = $1"
	row := r.db.QueryRowContext(ctx, query, id)

	var f FitProfile
	err := row.Scan(&f.Id, &f.Age, &f.Gender, &f.Height, &f.Weight, &f.ActivityLevel, &f.Goal, &f.Calories, &f.Protein, &f.Fat, &f.Carbs, &f.UserId, &f.CreatedAt, &f.UpdatedAt, &f.DeletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &f, nil
}
