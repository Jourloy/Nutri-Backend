package user

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	CreateUser(ctx context.Context, user *UserCreate) (*User, error)
	GetUser(ctx context.Context, id string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	IncreaseViewUpdates(ctx context.Context, uid string) (*User, error)
	UpdateLogin(ctx context.Context, uid string) error
	DeleteUser(ctx context.Context, id string) (*User, error)
	UpdateEmail(ctx context.Context, uid string, email string) (*User, error)
	SetEmailVerified(ctx context.Context, uid string, email string) (*User, error)
	InvalidateTokens(ctx context.Context, id string) error
	UpdateLocale(ctx context.Context, uid string, locale string) (*User, error)
	UpdateTimezone(ctx context.Context, uid string, timezone string) (*User, error)
	GetUserStats(ctx context.Context, uid string) (*UserStats, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

// единый список колонок — не используем SELECT *
const userColumns = `
    id, username, password_hash,
    email, email_verified, locale, timezone,
    is_accept_terms, is_accept_privacy, is_18, is_admin,
    token_version, view_updates, view_tutorial,
    logined_at, created_at, updated_at, deleted_at
`

func (r *repository) CreateUser(ctx context.Context, userCreate *UserCreate) (*User, error) {
	const insertQ = `
	INSERT INTO users (username, password_hash, view_updates, locale, timezone)
	VALUES (:username, :password_hash, :view_updates, :locale, :timezone)
	ON CONFLICT (username) DO NOTHING
	RETURNING ` + userColumns + `;`

	locale := userCreate.Locale
	timezone := userCreate.Timezone
	if timezone == nil {
		defaultTz := "UTC"
		timezone = &defaultTz
	}

	args := map[string]any{
		"username":      userCreate.Username,
		"password_hash": userCreate.PasswordHash,
		"view_updates":  3,
		"locale":        locale,
		"timezone":      timezone,
	}

	// Сначала пытаемся вставить и сразу вернуть строку
	rows, err := r.db.NamedQueryContext(ctx, insertQ, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var u User
		if err := rows.StructScan(&u); err != nil {
			return nil, err
		}
		return &u, nil
	}

	// Если вставка не произошла (конфликт) — читаем существующего по username
	const selectQ = `SELECT ` + userColumns + ` FROM users WHERE username = $1 LIMIT 1;`
	var u User
	if err := r.db.GetContext(ctx, &u, selectQ, userCreate.Username); err != nil {
		if err == sql.ErrNoRows {
			// Теоретически не должно случиться при ON CONFLICT, но вернём nil для явности
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *repository) GetUser(ctx context.Context, id string) (*User, error) {
	const q = `SELECT ` + userColumns + ` FROM users WHERE id = $1;`

	var u User
	if err := r.db.GetContext(ctx, &u, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *repository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	const q = `SELECT ` + userColumns + ` FROM users WHERE username = $1;`

	var u User
	if err := r.db.GetContext(ctx, &u, q, username); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *repository) IncreaseViewUpdates(ctx context.Context, uid string) (*User, error) {
	// Увеличиваем счётчик и возвращаем обновлённую строку
	const q = `
	UPDATE users
	SET view_updates = 3,
		updated_at   = now()
	WHERE id = $1
	RETURNING ` + userColumns + `;`

	var u User
	if err := r.db.GetContext(ctx, &u, q, uid); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *repository) UpdateLogin(ctx context.Context, uid string) error {
	const q = `
	UPDATE users
	SET logined_at = now(),
		updated_at  = now()
	WHERE id = $1
	RETURNING id;`

	var id string
	if err := r.db.GetContext(ctx, &id, q, uid); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	return nil
}

func (r *repository) DeleteUser(ctx context.Context, id string) (*User, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Удаляем все связанные записи вручную, так как большинство FK без ON DELETE CASCADE.
	deletions := []struct {
		query string
		args  []any
	}{
		{query: `DELETE FROM products WHERE user_id = $1`, args: []any{id}},
		{query: `DELETE FROM body_weights WHERE user_id = $1`, args: []any{id}},
		{query: `DELETE FROM body_measurements WHERE user_id = $1`, args: []any{id}},
		{query: `DELETE FROM body_plateau_events WHERE user_id = $1`, args: []any{id}},
		{query: `DELETE FROM body_activity WHERE user_id = $1`, args: []any{id}},
		{query: `DELETE FROM orders WHERE user_id = $1`, args: []any{id}},
		{query: `DELETE FROM user_achievements WHERE user_id = $1`, args: []any{id}},
		{query: `DELETE FROM telegram_profiles WHERE user_id = $1`, args: []any{id}},
		{query: `DELETE FROM coach_clients WHERE coach_user_id = $1 OR client_user_id = $1`, args: []any{id}},
		{query: `DELETE FROM subscriptions WHERE user_id = $1`, args: []any{id}},
		{query: `DELETE FROM fit_profiles WHERE user_id = $1`, args: []any{id}},
		{query: `DELETE FROM ai_user_limits WHERE user_id = $1`, args: []any{id}},
		{query: `DELETE FROM ticket_messages WHERE user_id = $1`, args: []any{id}},
		{query: `DELETE FROM ai_violations WHERE user_id = $1`, args: []any{id}},
		{query: `DELETE FROM ai_analysis_logs WHERE user_id = $1`, args: []any{id}},
		{query: `DELETE FROM consent_records WHERE user_id = $1`, args: []any{id}},
	}

	for _, d := range deletions {
		if _, err = tx.ExecContext(ctx, d.query, d.args...); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}

	const q = `DELETE FROM users WHERE id = $1 RETURNING ` + userColumns + `;`

	var u User
	if err = tx.GetContext(ctx, &u, q, id); err != nil {
		if err == sql.ErrNoRows {
			_ = tx.Rollback()
			return nil, nil
		}
		_ = tx.Rollback()
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *repository) InvalidateTokens(ctx context.Context, id string) error {
	const q = `
		UPDATE users
		SET token_version = token_version + 1,
			updated_at = now()
		WHERE id = $1
		RETURNING id;`

	var uid string
	if err := r.db.GetContext(ctx, &uid, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	return nil
}

func (r *repository) UpdateEmail(ctx context.Context, uid string, email string) (*User, error) {
	const q = `
        UPDATE users
        SET email = $2,
            email_verified = false,
            updated_at = now()
        WHERE id = $1
        RETURNING ` + userColumns + `;`

	var u User
	if err := r.db.GetContext(ctx, &u, q, uid, email); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *repository) SetEmailVerified(ctx context.Context, uid string, email string) (*User, error) {
	const q = `
        UPDATE users
        SET email = $2,
            email_verified = true,
            updated_at = now()
        WHERE id = $1
        RETURNING ` + userColumns + `;`

	var u User
	if err := r.db.GetContext(ctx, &u, q, uid, email); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *repository) UpdateLocale(ctx context.Context, uid string, locale string) (*User, error) {
	const q = `
        UPDATE users
        SET locale = $2,
            updated_at = now()
        WHERE id = $1
        RETURNING ` + userColumns + `;`

	var u User
	if err := r.db.GetContext(ctx, &u, q, uid, locale); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *repository) UpdateTimezone(ctx context.Context, uid string, timezone string) (*User, error) {
	const q = `
        UPDATE users
        SET timezone = $2,
            updated_at = now()
        WHERE id = $1
        RETURNING ` + userColumns + `;`

	var u User
	if err := r.db.GetContext(ctx, &u, q, uid, timezone); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *repository) GetUserStats(ctx context.Context, uid string) (*UserStats, error) {
	// First get user's timezone
	user, err := r.GetUser(ctx, uid)
	if err != nil {
		return nil, err
	}
	tz := "UTC"
	if user != nil && user.Timezone != nil && *user.Timezone != "" {
		tz = *user.Timezone
	}

	const q = `
	WITH user_data AS (
		SELECT
			COALESCE(EXTRACT(DAY FROM (NOW() - u.created_at))::int, 0) as days_since_registration,
			COALESCE(COUNT(DISTINCT p.logged_at)::int, 0) as days_with_logs,
			COALESCE(COUNT(p.id)::int, 0) as total_products
		FROM users u
		LEFT JOIN products p ON p.user_id = u.id
		WHERE u.id = $1
		GROUP BY u.created_at
	),
	achievements_data AS (
		SELECT COALESCE(COUNT(*)::int, 0) as unlocked_achievements
		FROM user_achievements
		WHERE user_id = $1
	),
	streak_data AS (
		SELECT date_trunc('day', logged_at)::date AS d
		FROM products
		WHERE user_id = $1
		AND logged_at >= CURRENT_DATE - INTERVAL '365 days'
		GROUP BY date_trunc('day', logged_at)::date
		ORDER BY d DESC
	)
	SELECT
		ud.days_since_registration,
		ud.days_with_logs,
		ud.total_products,
		ad.unlocked_achievements
	FROM user_data ud
	CROSS JOIN achievements_data ad;`

	var stats UserStats
	if err := r.db.QueryRowContext(ctx, q, uid).Scan(
		&stats.DaysSinceRegistration,
		&stats.DaysWithLogs,
		&stats.TotalProducts,
		&stats.UnlockedAchievements,
	); err != nil {
		if err == sql.ErrNoRows {
			return &UserStats{
				DaysSinceRegistration: 0,
				DaysWithLogs:          0,
				CurrentStreak:         0,
				TotalProducts:         0,
				UnlockedAchievements:  0,
			}, nil
		}
		return nil, err
	}

	// Calculate current streak separately (complex logic) using user's timezone
	stats.CurrentStreak = r.calculateCurrentStreak(ctx, uid, tz)

	return &stats, nil
}

// calculateCurrentStreak calculates the current consecutive days with at least 1 product
// Uses user's timezone to determine "today"
func (r *repository) calculateCurrentStreak(ctx context.Context, uid string, tz string) int {
	type row struct {
		D string `db:"d"`
	}
	var days []row

	// Get distinct days with products in the last 365 days
	const q = `
		SELECT DISTINCT date_trunc('day', logged_at)::date::text AS d
		FROM products
		WHERE user_id = $1
		AND logged_at >= $2::date - INTERVAL '365 days'
		ORDER BY d DESC`

	// Get today based on user's timezone
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	currentDate := time.Now().In(loc)
	todayStr := currentDate.Format("2006-01-02")

	if err := r.db.SelectContext(ctx, &days, q, uid, todayStr); err != nil || len(days) == 0 {
		return 0
	}

	// Build a map of dates
	dateMap := make(map[string]bool)
	for _, d := range days {
		dateMap[d.D] = true
	}

	// Count consecutive days from today backwards
	streak := 0

	for i := 0; i < 365; i++ {
		dateStr := currentDate.AddDate(0, 0, -i).Format("2006-01-02")
		if dateMap[dateStr] {
			streak++
		} else {
			break
		}
	}

	return streak
}
