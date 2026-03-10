package body

import (
	"context"
	"database/sql"
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
	GetPeriodDays(ctx context.Context, userId string, from, to time.Time) (map[string]bool, error)
	GetLatestGender(ctx context.Context, userId string) (string, error)
	GetLatestMacroGoals(ctx context.Context, userId string) (CycleGoals, error)

	// Activity CRUD
	CreateActivity(ctx context.Context, a ActivityCreate) (*Activity, error)
	UpdateActivity(ctx context.Context, a Activity) (*Activity, error)
	DeleteActivity(ctx context.Context, id int64, userId string) error
	GetActivity(ctx context.Context, userId string, from, to *time.Time) ([]Activity, error)

	// Workouts
	CreateWorkout(ctx context.Context, w WorkoutCreate) (*Workout, error)
	UpdateWorkout(ctx context.Context, w WorkoutCreate, id int64) (*Workout, error)
	DeleteWorkout(ctx context.Context, id int64, userId string) error
	GetWorkouts(ctx context.Context, userId string, from, to *time.Time) ([]Workout, error)
	GetWorkoutTotalDurationByDate(ctx context.Context, userId string, date time.Time) (int, error)
	GetLatestWaterLimit(ctx context.Context, userId string) (int, error)

	// Plateau events
	GetPlateauHistory(ctx context.Context, userId string, from, to *time.Time) ([]PlateauEvent, error)

	// Cycle tracking
	StartCycle(ctx context.Context, userId string, startDate time.Time) error
	StopCycle(ctx context.Context, userId string, endDate time.Time) (*Cycle, error)
	CreateHistoricalCycle(ctx context.Context, userId string, startDate, endDate time.Time) (*Cycle, error)
	GetOpenCycle(ctx context.Context, userId string) (*Cycle, error)
	GetCycles(ctx context.Context, userId string, from, to *time.Time, limit int) ([]Cycle, error)
	UpsertCycleDayLog(ctx context.Context, userId string, loggedAt time.Time, flowIntensity, note *string) (int64, error)
	ReplaceCycleDayEvents(ctx context.Context, dayLogId int64, events []CycleDayEventInput) error
	GetCycleDayLogs(ctx context.Context, userId string, from, to *time.Time) ([]CycleDayLog, error)
}

type repository struct{ db *sqlx.DB }

func NewRepository() Repository { return &repository{db: database.Database} }

// ===== Weights =====
func (r *repository) CreateWeight(ctx context.Context, w WeightCreate) (*Weight, error) {
	const q = `
        INSERT INTO body_weights (user_id, value, logged_at)
        VALUES (:user_id, :value, :logged_at)
        ON CONFLICT (user_id, logged_at) DO UPDATE SET value=EXCLUDED.value, updated_at=now()
        RETURNING id, user_id, value, logged_at, created_at, updated_at;`
	rows, err := r.db.NamedQueryContext(ctx, q, w)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out Weight
	if rows.Next() {
		if err := rows.StructScan(&out); err != nil {
			return nil, err
		}
	}
	return &out, nil
}

func (r *repository) UpdateWeight(ctx context.Context, w Weight) (*Weight, error) {
	const q = `
        UPDATE body_weights
        SET value=:value, logged_at=:logged_at, updated_at=now()
        WHERE id=:id AND user_id=:user_id
        RETURNING id, user_id, value, logged_at, created_at, updated_at;`
	rows, err := r.db.NamedQueryContext(ctx, q, w)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out Weight
	if rows.Next() {
		if err := rows.StructScan(&out); err != nil {
			return nil, err
		}
	}
	return &out, nil
}

func (r *repository) DeleteWeight(ctx context.Context, id int64, userId string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM body_weights WHERE id=$1 AND user_id=$2`, id, userId)
	return err
}

func (r *repository) GetWeights(ctx context.Context, userId string, from, to *time.Time) ([]Weight, error) {
	q := `SELECT id, user_id, value, logged_at, created_at, updated_at FROM body_weights WHERE user_id = $1`
	args := []any{userId}
	if from != nil {
		q += fmt.Sprintf(" AND logged_at >= $%d", len(args)+1)
		args = append(args, *from)
	}
	if to != nil {
		q += fmt.Sprintf(" AND logged_at <= $%d", len(args)+1)
		args = append(args, *to)
	}
	q += ` ORDER BY logged_at`
	var res []Weight
	if err := r.db.SelectContext(ctx, &res, q, args...); err != nil {
		return nil, err
	}
	return res, nil
}

func (r *repository) GetLatestWeight(ctx context.Context, userId string) (*Weight, error) {
	var w Weight
	err := r.db.GetContext(ctx, &w, `SELECT id, user_id, value, logged_at, created_at, updated_at FROM body_weights WHERE user_id=$1 ORDER BY logged_at DESC LIMIT 1`, userId)
	if err != nil {
		return nil, err
	}
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
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out Measurement
	if rows.Next() {
		if err := rows.StructScan(&out); err != nil {
			return nil, err
		}
	}
	return &out, nil
}

func (r *repository) UpdateMeasurement(ctx context.Context, m Measurement) (*Measurement, error) {
	const q = `
        UPDATE body_measurements
        SET chest=:chest, waist=:waist, hips=:hips, logged_at=:logged_at, updated_at=now()
        WHERE id=:id AND user_id=:user_id
        RETURNING id, user_id, chest, waist, hips, logged_at, created_at, updated_at;`
	rows, err := r.db.NamedQueryContext(ctx, q, m)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out Measurement
	if rows.Next() {
		if err := rows.StructScan(&out); err != nil {
			return nil, err
		}
	}
	return &out, nil
}

func (r *repository) DeleteMeasurement(ctx context.Context, id int64, userId string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM body_measurements WHERE id=$1 AND user_id=$2`, id, userId)
	return err
}

func (r *repository) GetMeasurements(ctx context.Context, userId string, from, to *time.Time) ([]Measurement, error) {
	q := `SELECT id, user_id, chest, waist, hips, logged_at, created_at, updated_at FROM body_measurements WHERE user_id = $1`
	args := []any{userId}
	if from != nil {
		q += fmt.Sprintf(" AND logged_at >= $%d", len(args)+1)
		args = append(args, *from)
	}
	if to != nil {
		q += fmt.Sprintf(" AND logged_at <= $%d", len(args)+1)
		args = append(args, *to)
	}
	q += ` ORDER BY logged_at`
	var res []Measurement
	if err := r.db.SelectContext(ctx, &res, q, args...); err != nil {
		return nil, err
	}
	return res, nil
}

func (r *repository) GetLatestMeasurement(ctx context.Context, userId string) (*Measurement, error) {
	var m Measurement
	err := r.db.GetContext(ctx, &m, `SELECT id, user_id, chest, waist, hips, logged_at, created_at, updated_at FROM body_measurements WHERE user_id=$1 ORDER BY logged_at DESC LIMIT 1`, userId)
	if err != nil {
		return nil, err
	}
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
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := map[string]float64{}
	for rows.Next() {
		var d time.Time
		var v float64
		if err := rows.Scan(&d, &v); err != nil {
			return nil, err
		}
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
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := map[string]float64{}
	for rows.Next() {
		var d time.Time
		var v float64
		if err := rows.Scan(&d, &v); err != nil {
			return nil, err
		}
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
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := map[string]int{}
	for rows.Next() {
		var d time.Time
		var v int
		if err := rows.Scan(&d, &v); err != nil {
			return nil, err
		}
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
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := map[string]int{}
	for rows.Next() {
		var d time.Time
		var v int
		if err := rows.Scan(&d, &v); err != nil {
			return nil, err
		}
		res[d.Format("2006-01-02")] = v
	}
	return res, rows.Err()
}

func (r *repository) GetPeriodDays(ctx context.Context, userId string, from, to time.Time) (map[string]bool, error) {
	rows, err := r.db.QueryxContext(ctx, `
        SELECT start_date, COALESCE(end_date, $3::date) AS end_date
        FROM body_cycles
        WHERE user_id = $1
          AND start_date <= $3
          AND (end_date IS NULL OR end_date >= $2)
        ORDER BY start_date DESC`, userId, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	periodDays := make(map[string]bool)
	for rows.Next() {
		var startDate time.Time
		var endDate time.Time
		if err := rows.Scan(&startDate, &endDate); err != nil {
			return nil, err
		}
		overlapStart := maxTime(startDate, from)
		overlapEnd := minTime(endDate, to)
		for d := overlapStart; !d.After(overlapEnd); d = d.AddDate(0, 0, 1) {
			periodDays[d.Format("2006-01-02")] = true
		}
	}
	return periodDays, rows.Err()
}

func (r *repository) GetLatestGender(ctx context.Context, userId string) (string, error) {
	var gender string
	err := r.db.GetContext(ctx, &gender, `
        SELECT gender
        FROM fit_profiles
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT 1`, userId)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return gender, nil
}

func (r *repository) GetLatestMacroGoals(ctx context.Context, userId string) (CycleGoals, error) {
	var goals CycleGoals
	err := r.db.GetContext(ctx, &goals, `
        SELECT calories::float AS calories, protein::float AS protein, fat::float AS fat, carbs::float AS carbs
        FROM fit_profiles
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT 1`, userId)
	if err != nil {
		if err == sql.ErrNoRows {
			return CycleGoals{}, nil
		}
		return CycleGoals{}, err
	}
	return goals, nil
}

// Activity CRUD
func (r *repository) CreateActivity(ctx context.Context, a ActivityCreate) (*Activity, error) {
	const q = `
        INSERT INTO body_activity (user_id, steps, sleep_min, logged_at)
        VALUES (:user_id, :steps, :sleep_min, :logged_at)
        ON CONFLICT (user_id, logged_at) DO UPDATE SET steps=COALESCE(EXCLUDED.steps, body_activity.steps), sleep_min=COALESCE(EXCLUDED.sleep_min, body_activity.sleep_min), updated_at=now()
        RETURNING id, user_id, steps, sleep_min, logged_at, created_at, updated_at;`
	rows, err := r.db.NamedQueryContext(ctx, q, a)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out Activity
	if rows.Next() {
		if err := rows.StructScan(&out); err != nil {
			return nil, err
		}
	}
	return &out, nil
}

func (r *repository) UpdateActivity(ctx context.Context, a Activity) (*Activity, error) {
	const q = `
        UPDATE body_activity
        SET steps=:steps, sleep_min=:sleep_min, logged_at=:logged_at, updated_at=now()
        WHERE id=:id AND user_id=:user_id
        RETURNING id, user_id, steps, sleep_min, logged_at, created_at, updated_at;`
	rows, err := r.db.NamedQueryContext(ctx, q, a)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out Activity
	if rows.Next() {
		if err := rows.StructScan(&out); err != nil {
			return nil, err
		}
	}
	return &out, nil
}

func (r *repository) DeleteActivity(ctx context.Context, id int64, userId string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM body_activity WHERE id=$1 AND user_id=$2`, id, userId)
	return err
}

func (r *repository) GetActivity(ctx context.Context, userId string, from, to *time.Time) ([]Activity, error) {
	q := `SELECT id, user_id, steps, sleep_min, logged_at, created_at, updated_at FROM body_activity WHERE user_id = $1`
	args := []any{userId}
	if from != nil {
		q += fmt.Sprintf(" AND logged_at >= $%d", len(args)+1)
		args = append(args, *from)
	}
	if to != nil {
		q += fmt.Sprintf(" AND logged_at <= $%d", len(args)+1)
		args = append(args, *to)
	}
	q += ` ORDER BY logged_at`
	var res []Activity
	if err := r.db.SelectContext(ctx, &res, q, args...); err != nil {
		return nil, err
	}
	return res, nil
}

// ===== Workouts =====
func (r *repository) CreateWorkout(ctx context.Context, w WorkoutCreate) (*Workout, error) {
	const q = `
		INSERT INTO body_workouts (user_id, logged_at, duration_min, workout_type, intensity, calories_burned, note)
		VALUES (:user_id, :logged_at, :duration_min, :workout_type, :intensity, :calories_burned, :note)
		RETURNING id, user_id, TO_CHAR(logged_at, 'YYYY-MM-DD') AS logged_at,
		          duration_min, workout_type, intensity, calories_burned, note, created_at, updated_at;`

	rows, err := r.db.NamedQueryContext(ctx, q, w)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out Workout
	if rows.Next() {
		if err := rows.StructScan(&out); err != nil {
			return nil, err
		}
	}
	return &out, nil
}

func (r *repository) UpdateWorkout(ctx context.Context, w WorkoutCreate, id int64) (*Workout, error) {
	payload := map[string]any{
		"id":              id,
		"user_id":         w.UserId,
		"logged_at":       w.LoggedAt,
		"duration_min":    w.DurationMin,
		"workout_type":    w.WorkoutType,
		"intensity":       w.Intensity,
		"calories_burned": w.CaloriesBurned,
		"note":            w.Note,
	}
	const q = `
		UPDATE body_workouts
		SET logged_at = :logged_at,
		    duration_min = :duration_min,
		    workout_type = :workout_type,
		    intensity = :intensity,
		    calories_burned = :calories_burned,
		    note = :note,
		    updated_at = NOW()
		WHERE id = :id AND user_id = :user_id
		RETURNING id, user_id, TO_CHAR(logged_at, 'YYYY-MM-DD') AS logged_at,
		          duration_min, workout_type, intensity, calories_burned, note, created_at, updated_at;`

	rows, err := r.db.NamedQueryContext(ctx, q, payload)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out Workout
	if rows.Next() {
		if err := rows.StructScan(&out); err != nil {
			return nil, err
		}
		return &out, nil
	}
	return nil, sql.ErrNoRows
}

func (r *repository) DeleteWorkout(ctx context.Context, id int64, userId string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM body_workouts WHERE id = $1 AND user_id = $2`, id, userId)
	return err
}

func (r *repository) GetWorkouts(ctx context.Context, userId string, from, to *time.Time) ([]Workout, error) {
	q := `SELECT id, user_id, TO_CHAR(logged_at, 'YYYY-MM-DD') AS logged_at,
	             duration_min, workout_type, intensity, calories_burned, note, created_at, updated_at
	      FROM body_workouts
	      WHERE user_id = $1`
	args := []any{userId}
	if from != nil {
		q += fmt.Sprintf(" AND logged_at >= $%d", len(args)+1)
		args = append(args, *from)
	}
	if to != nil {
		q += fmt.Sprintf(" AND logged_at <= $%d", len(args)+1)
		args = append(args, *to)
	}
	q += ` ORDER BY logged_at, created_at`

	var res []Workout
	if err := r.db.SelectContext(ctx, &res, q, args...); err != nil {
		return nil, err
	}
	return res, nil
}

func (r *repository) GetWorkoutTotalDurationByDate(ctx context.Context, userId string, date time.Time) (int, error) {
	var total int
	err := r.db.GetContext(ctx, &total, `
		SELECT COALESCE(SUM(duration_min), 0)
		FROM body_workouts
		WHERE user_id = $1 AND logged_at = $2::date`,
		userId, date)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return total, nil
}

func (r *repository) GetLatestWaterLimit(ctx context.Context, userId string) (int, error) {
	var waterLimit int
	err := r.db.GetContext(ctx, &waterLimit, `
		SELECT water_limit
		FROM fit_profiles
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1`, userId)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return waterLimit, nil
}

func (r *repository) GetPlateauHistory(ctx context.Context, userId string, from, to *time.Time) ([]PlateauEvent, error) {
	q := `SELECT id, user_id, window_start, window_end, goal, slope_weekly_pct, delta_kg, days_with_weight, calories_good_days, protein_good_days, window_days, is_plateau, reason, created_at FROM body_plateau_events WHERE user_id=$1`
	args := []any{userId}
	if from != nil {
		q += fmt.Sprintf(" AND window_start >= $%d", len(args)+1)
		args = append(args, *from)
	}
	if to != nil {
		q += fmt.Sprintf(" AND window_end <= $%d", len(args)+1)
		args = append(args, *to)
	}
	q += ` ORDER BY created_at DESC`
	var list []PlateauEvent
	if err := r.db.SelectContext(ctx, &list, q, args...); err != nil {
		return nil, err
	}
	return list, nil
}

func (r *repository) StartCycle(ctx context.Context, userId string, startDate time.Time) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `
        UPDATE body_cycles
        SET end_date = GREATEST(start_date, ($2::date - INTERVAL '1 day')::date),
            updated_at = now()
        WHERE user_id = $1
          AND end_date IS NULL`, userId, startDate); err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err = tx.ExecContext(ctx, `
        INSERT INTO body_cycles (user_id, start_date)
        VALUES ($1, $2::date)`, userId, startDate); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (r *repository) StopCycle(ctx context.Context, userId string, endDate time.Time) (*Cycle, error) {
	var cycle Cycle
	err := r.db.GetContext(ctx, &cycle, `
        UPDATE body_cycles
        SET end_date = $2::date,
            updated_at = now()
        WHERE id = (
            SELECT id
            FROM body_cycles
            WHERE user_id = $1 AND end_date IS NULL
            ORDER BY start_date DESC
            LIMIT 1
        )
        RETURNING id, user_id, start_date, end_date, created_at, updated_at`, userId, endDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &cycle, nil
}

func (r *repository) CreateHistoricalCycle(
	ctx context.Context,
	userId string,
	startDate, endDate time.Time,
) (*Cycle, error) {
	var cycle Cycle
	err := r.db.GetContext(ctx, &cycle, `
        INSERT INTO body_cycles (user_id, start_date, end_date)
        VALUES ($1, $2::date, $3::date)
        RETURNING id, user_id, start_date, end_date, created_at, updated_at`,
		userId, startDate, endDate)
	if err != nil {
		return nil, err
	}
	return &cycle, nil
}

func (r *repository) GetOpenCycle(ctx context.Context, userId string) (*Cycle, error) {
	var cycle Cycle
	err := r.db.GetContext(ctx, &cycle, `
        SELECT id, user_id, start_date, end_date, created_at, updated_at
        FROM body_cycles
        WHERE user_id = $1 AND end_date IS NULL
        ORDER BY start_date DESC
        LIMIT 1`, userId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &cycle, nil
}

func (r *repository) GetCycles(ctx context.Context, userId string, from, to *time.Time, limit int) ([]Cycle, error) {
	q := `SELECT id, user_id, start_date, end_date, created_at, updated_at FROM body_cycles WHERE user_id = $1`
	args := []any{userId}
	if from != nil {
		q += fmt.Sprintf(" AND start_date >= $%d", len(args)+1)
		args = append(args, *from)
	}
	if to != nil {
		q += fmt.Sprintf(" AND start_date <= $%d", len(args)+1)
		args = append(args, *to)
	}
	q += ` ORDER BY start_date DESC`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT $%d", len(args)+1)
		args = append(args, limit)
	}
	var cycles []Cycle
	if err := r.db.SelectContext(ctx, &cycles, q, args...); err != nil {
		return nil, err
	}
	return cycles, nil
}

func (r *repository) UpsertCycleDayLog(
	ctx context.Context,
	userId string,
	loggedAt time.Time,
	flowIntensity, note *string,
) (int64, error) {
	var id int64
	err := r.db.GetContext(ctx, &id, `
        INSERT INTO body_cycle_daily_logs (user_id, logged_at, flow_intensity, note)
        VALUES ($1, $2::date, $3, $4)
        ON CONFLICT (user_id, logged_at)
        DO UPDATE SET
            flow_intensity = EXCLUDED.flow_intensity,
            note = EXCLUDED.note,
            updated_at = now()
        RETURNING id`, userId, loggedAt, flowIntensity, note)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *repository) ReplaceCycleDayEvents(ctx context.Context, dayLogId int64, events []CycleDayEventInput) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM body_cycle_daily_events WHERE day_log_id = $1`, dayLogId); err != nil {
		_ = tx.Rollback()
		return err
	}

	for _, e := range events {
		quantity := e.Quantity
		if quantity <= 0 {
			quantity = 1
		}
		if _, err = tx.ExecContext(ctx, `
            INSERT INTO body_cycle_daily_events (day_log_id, event_category, event_code, quantity, intensity)
            VALUES ($1, $2, $3, $4, $5)`,
			dayLogId, e.EventCategory, e.EventCode, quantity, e.Intensity); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (r *repository) GetCycleDayLogs(ctx context.Context, userId string, from, to *time.Time) ([]CycleDayLog, error) {
	q := `
        SELECT
            l.id AS day_log_id,
            l.logged_at,
            l.flow_intensity,
            l.note,
            e.id AS event_id,
            e.event_category,
            e.event_code,
            e.quantity,
            e.intensity
        FROM body_cycle_daily_logs l
        LEFT JOIN body_cycle_daily_events e ON e.day_log_id = l.id
        WHERE l.user_id = $1`
	args := []any{userId}
	if from != nil {
		q += fmt.Sprintf(" AND l.logged_at >= $%d", len(args)+1)
		args = append(args, *from)
	}
	if to != nil {
		q += fmt.Sprintf(" AND l.logged_at <= $%d", len(args)+1)
		args = append(args, *to)
	}
	q += ` ORDER BY l.logged_at, e.id`

	type row struct {
		DayLogID      int64          `db:"day_log_id"`
		LoggedAt      time.Time      `db:"logged_at"`
		FlowIntensity sql.NullString `db:"flow_intensity"`
		Note          sql.NullString `db:"note"`
		EventID       sql.NullInt64  `db:"event_id"`
		EventCategory sql.NullString `db:"event_category"`
		EventCode     sql.NullString `db:"event_code"`
		Quantity      sql.NullInt64  `db:"quantity"`
		Intensity     sql.NullString `db:"intensity"`
	}

	rows, err := r.db.QueryxContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logsByID := make(map[int64]*CycleDayLog)
	order := make([]int64, 0)
	for rows.Next() {
		var rrow row
		if err := rows.StructScan(&rrow); err != nil {
			return nil, err
		}
		logItem, exists := logsByID[rrow.DayLogID]
		if !exists {
			logItem = &CycleDayLog{
				Id:       rrow.DayLogID,
				LoggedAt: rrow.LoggedAt.Format("2006-01-02"),
				Events:   []CycleDailyEvent{},
			}
			if rrow.FlowIntensity.Valid {
				v := rrow.FlowIntensity.String
				logItem.FlowIntensity = &v
			}
			if rrow.Note.Valid {
				v := rrow.Note.String
				logItem.Note = &v
			}
			logsByID[rrow.DayLogID] = logItem
			order = append(order, rrow.DayLogID)
		}
		if rrow.EventID.Valid {
			var intensity *string
			if rrow.Intensity.Valid {
				v := rrow.Intensity.String
				intensity = &v
			}
			quantity := 1
			if rrow.Quantity.Valid && rrow.Quantity.Int64 > 0 {
				quantity = int(rrow.Quantity.Int64)
			}
			logItem.Events = append(logItem.Events, CycleDailyEvent{
				Id:            rrow.EventID.Int64,
				EventCategory: rrow.EventCategory.String,
				EventCode:     rrow.EventCode.String,
				Quantity:      quantity,
				Intensity:     intensity,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	logs := make([]CycleDayLog, 0, len(order))
	for _, id := range order {
		logs = append(logs, *logsByID[id])
	}
	return logs, nil
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
