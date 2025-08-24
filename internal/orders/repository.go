package orders

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri-backend/internal/database"
	"github.com/jourloy/nutri-backend/internal/plan"
)

type Repository interface {
	Create(ctx context.Context, oc OrderCreate) (*Order, error)
	UpdatePayment(ctx context.Context, id int64, paymentId, paymentURL, status string) error
	FindPlan(ctx context.Context, name, period string) (*plan.Plan, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

func (r *repository) Create(ctx context.Context, oc OrderCreate) (*Order, error) {
	const q = `
    INSERT INTO orders (
        user_id, plan_id, amount_minor, currency, status
    ) VALUES (
        :user_id, :plan_id, :amount_minor, :currency, :status
    )
    RETURNING id, user_id, plan_id, amount_minor, currency, status, payment_id, payment_url, created_at, updated_at;`

	rows, err := r.db.NamedQueryContext(ctx, q, oc)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var o Order
		if err := rows.StructScan(&o); err != nil {
			return nil, err
		}
		return &o, nil
	}
	return nil, nil
}

func (r *repository) UpdatePayment(ctx context.Context, id int64, paymentId, paymentURL, status string) error {
	const q = `
    UPDATE orders SET
        payment_id = $1,
        payment_url = $2,
        status = $3,
        updated_at = now()
    WHERE id = $4;`
	_, err := r.db.ExecContext(ctx, q, paymentId, paymentURL, status, id)
	return err
}

func (r *repository) FindPlan(ctx context.Context, name, period string) (*plan.Plan, error) {
	const q = `
    SELECT id, code, name, plan_type, version, currency, amount_minor, billing_period,
           trial_days, client_limit, is_active, created_at, updated_at, external_product_id, external_price_id
    FROM plans
    WHERE name = $1 AND billing_period = $2 AND is_active = TRUE
    LIMIT 1;`
	var p plan.Plan
	if err := r.db.GetContext(ctx, &p, q, name, period); err != nil {
		return nil, err
	}
	return &p, nil
}
