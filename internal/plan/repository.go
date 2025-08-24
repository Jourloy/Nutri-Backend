package plan

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	GetAllActive(ctx context.Context) ([]Plan, error)
	Create(ctx context.Context, pc PlanCreate) (*Plan, error)
	Update(ctx context.Context, p Plan) (*Plan, error)
	Delete(ctx context.Context, id int64) error
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

func (r *repository) GetAllActive(ctx context.Context) ([]Plan, error) {
	const q = `
        SELECT id, code, name, plan_type, version, currency,
               amount_minor, billing_period, trial_days, client_limit,
               is_active, created_at, updated_at, external_product_id, external_price_id
        FROM plans
        WHERE is_active = TRUE
        ORDER BY amount_minor ASC`

	var ps []Plan
	if err := r.db.SelectContext(ctx, &ps, q); err != nil {
		return nil, err
	}
	return ps, nil
}

func (r *repository) Create(ctx context.Context, pc PlanCreate) (*Plan, error) {
	const q = `
        INSERT INTO plans (
                code, name, plan_type, version, currency, amount_minor,
                billing_period, trial_days, client_limit, is_active,
                external_product_id, external_price_id
        ) VALUES (
                :code, :name, :plan_type, :version, :currency, :amount_minor,
                :billing_period, :trial_days, :client_limit, :is_active,
                :external_product_id, :external_price_id
        )
        RETURNING id, code, name, plan_type, version, currency, amount_minor,
                  billing_period, trial_days, client_limit, is_active,
                  created_at, updated_at, external_product_id, external_price_id;`

	rows, err := r.db.NamedQueryContext(ctx, q, pc)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var p Plan
		if err := rows.StructScan(&p); err != nil {
			return nil, err
		}
		return &p, nil
	}
	return nil, nil
}

func (r *repository) Update(ctx context.Context, p Plan) (*Plan, error) {
	const q = `
        UPDATE plans SET
                code = :code,
                name = :name,
                plan_type = :plan_type,
                version = :version,
                currency = :currency,
                amount_minor = :amount_minor,
                billing_period = :billing_period,
                trial_days = :trial_days,
                client_limit = :client_limit,
                is_active = :is_active,
                external_product_id = :external_product_id,
                external_price_id = :external_price_id,
                updated_at = now()
        WHERE id = :id
        RETURNING id, code, name, plan_type, version, currency, amount_minor,
                  billing_period, trial_days, client_limit, is_active,
                  created_at, updated_at, external_product_id, external_price_id;`

	rows, err := r.db.NamedQueryContext(ctx, q, p)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var res Plan
		if err := rows.StructScan(&res); err != nil {
			return nil, err
		}
		return &res, nil
	}
	return nil, nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	const q = `DELETE FROM plans WHERE id = $1 RETURNING id;`

	var deletedID int64
	if err := r.db.GetContext(ctx, &deletedID, q, id); err != nil {
		return err
	}
	return nil
}
