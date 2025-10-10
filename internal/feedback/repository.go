package feedback

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	Create(ctx context.Context, feedback Feedback) (*Feedback, error)
	GetLatestByUser(ctx context.Context, userID string) (*Feedback, error)
	UpdateViewed(ctx context.Context, id string, viewed bool) (*Feedback, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

func (r *repository) Create(ctx context.Context, feedback Feedback) (*Feedback, error) {
	const query = `
        INSERT INTO feedbacks (user_id, status, message)
        VALUES (:user_id, :status, :message)
        RETURNING id, user_id, status, message, viewed, created_at;
    `

	rows, err := r.db.NamedQueryContext(ctx, query, feedback)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var res Feedback
		if err := rows.StructScan(&res); err != nil {
			return nil, err
		}
		return &res, nil
	}

	return nil, nil
}

func (r *repository) GetLatestByUser(ctx context.Context, userID string) (*Feedback, error) {
	const query = `
        SELECT id, user_id, status, message, viewed, created_at
        FROM feedbacks
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT 1;
    `

	var res Feedback
	if err := r.db.GetContext(ctx, &res, query, userID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &res, nil
}

func (r *repository) UpdateViewed(ctx context.Context, id string, viewed bool) (*Feedback, error) {
	const query = `
        UPDATE feedbacks
        SET viewed = $2
        WHERE id = $1
        RETURNING id, user_id, status, message, viewed, created_at;
    `

	var res Feedback
	if err := r.db.GetContext(ctx, &res, query, id, viewed); err != nil {
		return nil, err
	}
	return &res, nil
}
