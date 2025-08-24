package subscription

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	Create(ctx context.Context, sc SubscriptionCreate) (*Subscription, error)
	Update(ctx context.Context, s Subscription) (*Subscription, error)
	Delete(ctx context.Context, id int64, uid string) error
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

func (r *repository) Create(ctx context.Context, sc SubscriptionCreate) (*Subscription, error) {
	const q = `
        INSERT INTO subscriptions (
                user_id, plan_id, status, period_start, period_end,
                cancel_at, canceled_at, trial_end, amount_minor, currency,
                billing_period, external_subscription_id, external_customer_id
        ) VALUES (
                :user_id, :plan_id, :status, :period_start, :period_end,
                :cancel_at, :canceled_at, :trial_end, :amount_minor, :currency,
                :billing_period, :external_subscription_id, :external_customer_id
        )
        RETURNING id, user_id, plan_id, status, period_start, period_end,
                  cancel_at, canceled_at, trial_end, amount_minor, currency,
                  billing_period, external_subscription_id, external_customer_id,
                  created_at, updated_at;`

	rows, err := r.db.NamedQueryContext(ctx, q, sc)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var s Subscription
		if err := rows.StructScan(&s); err != nil {
			return nil, err
		}
		return &s, nil
	}
	return nil, nil
}

func (r *repository) Update(ctx context.Context, s Subscription) (*Subscription, error) {
	const q = `
        UPDATE subscriptions SET
                plan_id = :plan_id,
                status = :status,
                period_start = :period_start,
                period_end = :period_end,
                cancel_at = :cancel_at,
                canceled_at = :canceled_at,
                trial_end = :trial_end,
                amount_minor = :amount_minor,
                currency = :currency,
                billing_period = :billing_period,
                external_subscription_id = :external_subscription_id,
                external_customer_id = :external_customer_id,
                updated_at = now()
        WHERE id = :id AND user_id = :user_id
        RETURNING id, user_id, plan_id, status, period_start, period_end,
                  cancel_at, canceled_at, trial_end, amount_minor, currency,
                  billing_period, external_subscription_id, external_customer_id,
                  created_at, updated_at;`

	rows, err := r.db.NamedQueryContext(ctx, q, s)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var res Subscription
		if err := rows.StructScan(&res); err != nil {
			return nil, err
		}
		return &res, nil
	}
	return nil, nil
}

func (r *repository) Delete(ctx context.Context, id int64, uid string) error {
	const q = `DELETE FROM subscriptions WHERE id = $1 AND user_id = $2 RETURNING id;`

	var deletedID int64
	if err := r.db.GetContext(ctx, &deletedID, q, id, uid); err != nil {
		return err
	}
	return nil
}
