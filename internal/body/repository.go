package body

import (
    "context"
    "fmt"
    "time"

    "github.com/jmoiron/sqlx"

    "github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
    // Weights
    CreateWeight(ctx context.Context, w WeightCreate) (*Weight, error)
    UpdateWeight(ctx context.Context, w Weight) (*Weight, error)
    DeleteWeight(ctx context.Context, id int64, userId string) error
    GetWeights(ctx context.Context, userId string, from, to *time.Time) ([]Weight, error)
    GetLatestWeight(ctx context.Context, userId string) (*Weight, error)

    // Measurements
    CreateMeasurement(ctx context.Context, m MeasurementCreate) (*Measurement, error)
    UpdateMeasurement(ctx context.Context, m Measurement) (*Measurement, error)
    DeleteMeasurement(ctx context.Context, id int64, userId string) error
    GetMeasurements(ctx context.Context, userId string, from, to *time.Time) ([]Measurement, error)
    GetLatestMeasurement(ctx context.Context, userId string) (*Measurement, error)

    // Analytics helpers
    GetDailyCalories(ctx context.Context, userId string, from, to time.Time) (map[string]float64, error)
    GetDailyProtein(ctx context.Context, userId string, from, to time.Time) (map[string]float64, error)
    GetDailySteps(ctx context.Context, userId string, from, to time.Time) (map[string]int, error)
    GetDailySleepMin(ctx context.Context, userId string, from, to time.Time) (map[string]int, error)

    // Activity CRUD
    CreateActivity(ctx context.Context, a ActivityCreate) (*Activity, error)
    UpdateActivity(ctx context.Context, a Activity) (*Activity, error)
    DeleteActivity(ctx context.Context, id int64, userId string) error
    GetActivity(ctx context.Context, userId string, from, to *time.Time) ([]Activity, error)

    // Plateau events
    GetPlateauHistory(ctx context.Context, userId string, from, to *time.Time) ([]PlateauEvent, error)
}

type repository struct { db *sqlx.DB }

func NewRepository() Repository { return &repository{db: database.Database} }

// ===== Weights =====
func (r *repository) CreateWeight(ctx context.Context, w WeightCreate) (*Weight, error) {
    const q = `
        INSERT INTO body_weights (user_id, value, logged_at)
        VALUES (:user_id, :value, :logged_at)
        ON CONFLICT (user_id, logged_at) DO UPDATE SET value=EXCLUDED.value, updated_at=now()
        RETURNING id, user_id, value, logged_at, created_at, updated_at;`
    rows, err := r.db.NamedQueryContext(ctx, q, w)
    if err != nil { return nil, err }
    defer rows.Close()
    var out Weight
    if rows.Next() { if err := rows.StructScan(&out); err != nil { return nil, err } }
    return &out, nil
}

func (r *repository) UpdateWeight(ctx context.Context, w Weight) (*Weight, error) {
    const q = `
        UPDATE body_weights
        SET value=:value, logged_at=:logged_at, updated_at=now()
        WHERE id=:id AND user_id=:user_id
        RETURNING id, user_id, value, logged_at, created_at, updated_at;`
    rows, err := r.db.NamedQueryContext(ctx, q, w)
    if err != nil { return nil, err }
    defer rows.Close()
    var out Weight
    if rows.Next() { if err := rows.StructScan(&out); err != nil { return nil, err } }
    return &out, nil
}

func (r *repository) DeleteWeight(ctx context.Context, id int64, userId string) error {
    _, err := r.db.ExecContext(ctx, `DELETE FROM body_weights WHERE id=$1 AND user_id=$2`, id, userId)
    return err
}

func (r *repository) GetWeights(ctx context.Context, userId string, from, to *time.Time) ([]Weight, error) {
    q := `SELECT id, user_id, value, logged_at, created_at, updated_at FROM body_weights WHERE user_id = $1`
    args := []any{userId}
    if from != nil { q += fmt.Sprintf(" AND logged_at >= $%d", len(args)+1); args = append(args, *from) }
    if to != nil { q += fmt.Sprintf(" AND logged_at <= $%d", len(args)+1); args = append(args, *to) }
    q += ` ORDER BY logged_at`
    var res []Weight
    if err := r.db.SelectContext(ctx, &res, q, args...); err != nil { return nil, err }
    return res, nil
}

func (r *repository) GetLatestWeight(ctx context.Context, userId string) (*Weight, error) {
    var w Weight
    err := r.db.GetContext(ctx, &w, `SELECT id, user_id, value, logged_at, created_at, updated_at FROM body_weights WHERE user_id=$1 ORDER BY logged_at DESC LIMIT 1`, userId)
    if err != nil { return nil, err }
    return &w, nil
}

// ===== Measurements =====
func (r *repository) CreateMeasurement(ctx context.Context, m MeasurementCreate) (*Measurement, error) {
    const q = `
        INSERT INTO body_measurements (user_id, chest, waist, hips, logged_at)
        VALUES (:user_id, :chest, :waist, :hips, :logged_at)
        ON CONFLICT (user_id, logged_at) DO UPDATE SET chest=EXCLUDED.chest, waist=EXCLUDED.waist, hips=EXCLUDED.hips, updated_at=now()
        RETURNING id, user_id, chest, waist, hips, logged_at, created_at, updated_at;`
    rows, err := r.db.NamedQueryContext(ctx, q, m)
    if err != nil { return nil, err }
    defer rows.Close()
    var out Measurement
    if rows.Next() { if err := rows.StructScan(&out); err != nil { return nil, err } }
    return &out, nil
}

func (r *repository) UpdateMeasurement(ctx context.Context, m Measurement) (*Measurement, error) {
    const q = `
        UPDATE body_measurements
        SET chest=:chest, waist=:waist, hips=:hips, logged_at=:logged_at, updated_at=now()
        WHERE id=:id AND user_id=:user_id
        RETURNING id, user_id, chest, waist, hips, logged_at, created_at, updated_at;`
    rows, err := r.db.NamedQueryContext(ctx, q, m)
    if err != nil { return nil, err }
    defer rows.Close()
    var out Measurement
    if rows.Next() { if err := rows.StructScan(&out); err != nil { return nil, err } }
    return &out, nil
}

func (r *repository) DeleteMeasurement(ctx context.Context, id int64, userId string) error {
    _, err := r.db.ExecContext(ctx, `DELETE FROM body_measurements WHERE id=$1 AND user_id=$2`, id, userId)
    return err
}

func (r *repository) GetMeasurements(ctx context.Context, userId string, from, to *time.Time) ([]Measurement, error) {
    q := `SELECT id, user_id, chest, waist, hips, logged_at, created_at, updated_at FROM body_measurements WHERE user_id = $1`
    args := []any{userId}
    if from != nil { q += fmt.Sprintf(" AND logged_at >= $%d", len(args)+1); args = append(args, *from) }
    if to != nil { q += fmt.Sprintf(" AND logged_at <= $%d", len(args)+1); args = append(args, *to) }
    q += ` ORDER BY logged_at`
    var res []Measurement
    if err := r.db.SelectContext(ctx, &res, q, args...); err != nil { return nil, err }
    return res, nil
}

func (r *repository) GetLatestMeasurement(ctx context.Context, userId string) (*Measurement, error) {
    var m Measurement
    err := r.db.GetContext(ctx, &m, `SELECT id, user_id, chest, waist, hips, logged_at, created_at, updated_at FROM body_measurements WHERE user_id=$1 ORDER BY logged_at DESC LIMIT 1`, userId)
    if err != nil { return nil, err }
    return &m, nil
}

// ===== Analytics helpers from products =====
func (r *repository) GetDailyCalories(ctx context.Context, userId string, from, to time.Time) (map[string]float64, error) {
    rows, err := r.db.QueryxContext(ctx, `
        SELECT created_at::date AS d, COALESCE(SUM(calories),0)::float AS v
        FROM products
        WHERE user_id=$1 AND created_at >= $2 AND created_at < ($3 + INTERVAL '1 day')
        GROUP BY d
        ORDER BY d`, userId, from, to)
    if err != nil { return nil, err }
    defer rows.Close()
    res := map[string]float64{}
    for rows.Next() {
        var d time.Time
        var v float64
        if err := rows.Scan(&d, &v); err != nil { return nil, err }
        res[d.Format("2006-01-02")] = v
    }
    return res, rows.Err()
}

func (r *repository) GetDailyProtein(ctx context.Context, userId string, from, to time.Time) (map[string]float64, error) {
    rows, err := r.db.QueryxContext(ctx, `
        SELECT created_at::date AS d, COALESCE(SUM(protein),0)::float AS v
        FROM products
        WHERE user_id=$1 AND created_at >= $2 AND created_at < ($3 + INTERVAL '1 day')
        GROUP BY d
        ORDER BY d`, userId, from, to)
    if err != nil { return nil, err }
    defer rows.Close()
    res := map[string]float64{}
    for rows.Next() {
        var d time.Time
        var v float64
        if err := rows.Scan(&d, &v); err != nil { return nil, err }
        res[d.Format("2006-01-02")] = v
    }
    return res, rows.Err()
}

func (r *repository) GetDailySteps(ctx context.Context, userId string, from, to time.Time) (map[string]int, error) {
    rows, err := r.db.QueryxContext(ctx, `
        SELECT logged_at AS d, COALESCE(steps,0) AS v
        FROM body_activity
        WHERE user_id=$1 AND logged_at >= $2 AND logged_at <= $3
        ORDER BY d`, userId, from, to)
    if err != nil { return nil, err }
    defer rows.Close()
    res := map[string]int{}
    for rows.Next() {
        var d time.Time
        var v int
        if err := rows.Scan(&d, &v); err != nil { return nil, err }
        res[d.Format("2006-01-02")] = v
    }
    return res, rows.Err()
}

func (r *repository) GetDailySleepMin(ctx context.Context, userId string, from, to time.Time) (map[string]int, error) {
    rows, err := r.db.QueryxContext(ctx, `
        SELECT logged_at AS d, COALESCE(sleep_min,0) AS v
        FROM body_activity
        WHERE user_id=$1 AND logged_at >= $2 AND logged_at <= $3
        ORDER BY d`, userId, from, to)
    if err != nil { return nil, err }
    defer rows.Close()
    res := map[string]int{}
    for rows.Next() {
        var d time.Time
        var v int
        if err := rows.Scan(&d, &v); err != nil { return nil, err }
        res[d.Format("2006-01-02")] = v
    }
    return res, rows.Err()
}

// Activity CRUD
func (r *repository) CreateActivity(ctx context.Context, a ActivityCreate) (*Activity, error) {
    const q = `
        INSERT INTO body_activity (user_id, steps, sleep_min, logged_at)
        VALUES (:user_id, :steps, :sleep_min, :logged_at)
        ON CONFLICT (user_id, logged_at) DO UPDATE SET steps=COALESCE(EXCLUDED.steps, body_activity.steps), sleep_min=COALESCE(EXCLUDED.sleep_min, body_activity.sleep_min), updated_at=now()
        RETURNING id, user_id, steps, sleep_min, logged_at, created_at, updated_at;`
    rows, err := r.db.NamedQueryContext(ctx, q, a)
    if err != nil { return nil, err }
    defer rows.Close()
    var out Activity
    if rows.Next() { if err := rows.StructScan(&out); err != nil { return nil, err } }
    return &out, nil
}

func (r *repository) UpdateActivity(ctx context.Context, a Activity) (*Activity, error) {
    const q = `
        UPDATE body_activity
        SET steps=:steps, sleep_min=:sleep_min, logged_at=:logged_at, updated_at=now()
        WHERE id=:id AND user_id=:user_id
        RETURNING id, user_id, steps, sleep_min, logged_at, created_at, updated_at;`
    rows, err := r.db.NamedQueryContext(ctx, q, a)
    if err != nil { return nil, err }
    defer rows.Close()
    var out Activity
    if rows.Next() { if err := rows.StructScan(&out); err != nil { return nil, err } }
    return &out, nil
}

func (r *repository) DeleteActivity(ctx context.Context, id int64, userId string) error {
    _, err := r.db.ExecContext(ctx, `DELETE FROM body_activity WHERE id=$1 AND user_id=$2`, id, userId)
    return err
}

func (r *repository) GetActivity(ctx context.Context, userId string, from, to *time.Time) ([]Activity, error) {
    q := `SELECT id, user_id, steps, sleep_min, logged_at, created_at, updated_at FROM body_activity WHERE user_id = $1`
    args := []any{userId}
    if from != nil { q += fmt.Sprintf(" AND logged_at >= $%d", len(args)+1); args = append(args, *from) }
    if to != nil { q += fmt.Sprintf(" AND logged_at <= $%d", len(args)+1); args = append(args, *to) }
    q += ` ORDER BY logged_at`
    var res []Activity
    if err := r.db.SelectContext(ctx, &res, q, args...); err != nil { return nil, err }
    return res, nil
}

func (r *repository) GetPlateauHistory(ctx context.Context, userId string, from, to *time.Time) ([]PlateauEvent, error) {
    q := `SELECT id, user_id, window_start, window_end, goal, slope_weekly_pct, delta_kg, days_with_weight, calories_good_days, protein_good_days, window_days, is_plateau, reason, created_at FROM body_plateau_events WHERE user_id=$1`
    args := []any{userId}
    if from != nil { q += fmt.Sprintf(" AND window_start >= $%d", len(args)+1); args = append(args, *from) }
    if to != nil { q += fmt.Sprintf(" AND window_end <= $%d", len(args)+1); args = append(args, *to) }
    q += ` ORDER BY created_at DESC`
    var list []PlateauEvent
    if err := r.db.SelectContext(ctx, &list, q, args...); err != nil { return nil, err }
    return list, nil
}

//
