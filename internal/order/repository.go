package order

import (
    "context"

    "github.com/jmoiron/sqlx"
    "github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
    Create(ctx context.Context, o Order) (*Order, error)
    Update(ctx context.Context, o Order) (*Order, error)
    GetByTbOrderId(ctx context.Context, tbOrderId string) (*Order, error)
    GetAll(ctx context.Context, userID string, isAdmin bool) ([]Order, error)
    Delete(ctx context.Context, id int64, userID string, isAdmin bool) error
}

type repository struct { db *sqlx.DB }

func NewRepository() Repository { return &repository{db: database.Database} }

const columns = `id, status, user_id, plan_id, amount_minor, currency, tb_order_id, tb_rebill_id, payment_url, paid_at, last_error, created_at, updated_at`

func (r *repository) Create(ctx context.Context, o Order) (*Order, error) {
    const q = `INSERT INTO orders (status, user_id, plan_id, amount_minor, currency, tb_order_id, tb_rebill_id, payment_url, paid_at, last_error)
VALUES (:status,:user_id,:plan_id,:amount_minor,:currency,:tb_order_id,:tb_rebill_id,:payment_url,:paid_at,:last_error)
RETURNING ` + columns + `;`
    rows, err := r.db.NamedQueryContext(ctx, q, o)
    if err != nil { return nil, err }
    defer rows.Close()
    if rows.Next() { var res Order; if err := rows.StructScan(&res); err != nil { return nil, err }; return &res, nil }
    return nil, nil
}

func (r *repository) Update(ctx context.Context, o Order) (*Order, error) {
    const q = `UPDATE orders SET status=:status, user_id=:user_id, plan_id=:plan_id, amount_minor=:amount_minor, currency=:currency, tb_order_id=:tb_order_id, tb_rebill_id=:tb_rebill_id, payment_url=:payment_url, paid_at=:paid_at, last_error=:last_error, updated_at=now()
WHERE id=:id RETURNING ` + columns + `;`
    rows, err := r.db.NamedQueryContext(ctx, q, o)
    if err != nil { return nil, err }
    defer rows.Close()
    if rows.Next() { var res Order; if err := rows.StructScan(&res); err != nil { return nil, err }; return &res, nil }
    return nil, nil
}

func (r *repository) GetByTbOrderId(ctx context.Context, tbOrderId string) (*Order, error) {
    const q = `SELECT ` + columns + ` FROM orders WHERE tb_order_id=$1`
    var o Order
    if err := r.db.GetContext(ctx, &o, q, tbOrderId); err != nil { return nil, err }
    return &o, nil
}

func (r *repository) GetAll(ctx context.Context, userID string, isAdmin bool) ([]Order, error) {
    q := `SELECT ` + columns + ` FROM orders`
    args := []any{}
    if !isAdmin || userID != "" { q += ` WHERE user_id=$1`; args = append(args, userID) }
    var res []Order
    if err := r.db.SelectContext(ctx, &res, q, args...); err != nil { return nil, err }
    return res, nil
}

func (r *repository) Delete(ctx context.Context, id int64, userID string, isAdmin bool) error {
    q := `DELETE FROM orders WHERE id=$1`
    args := []any{id}
    if !isAdmin { q += ` AND user_id=$2`; args = append(args, userID) }
    _, err := r.db.ExecContext(ctx, q, args...)
    return err
}

