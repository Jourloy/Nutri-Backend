package ai

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jourloy/nutri-backend/internal/database"
)

type Repository interface {
	// Analysis logs
	CreateAnalysisLog(ctx context.Context, log AnalysisLog) (*AnalysisLog, error)
	UpdateAnalysisLog(ctx context.Context, log AnalysisLog) error
	GetAnalysisLogById(ctx context.Context, id int64) (*AnalysisLog, error)
	GetUserAnalysisLogs(ctx context.Context, userId string, limit int) ([]AnalysisLog, error)

	// User limits
	GetOrCreateUserLimit(ctx context.Context, userId, requestType string, date time.Time) (*UserLimit, error)
	IncrementUserLimit(ctx context.Context, userId, requestType string, date time.Time) error
	GetUserLimit(ctx context.Context, userId, requestType string, date time.Time) (*UserLimit, error)

	// Violations
	CreateViolation(ctx context.Context, violation Violation) (*Violation, error)
	GetUserActiveViolations(ctx context.Context, userId string) ([]Violation, error)
	GetUnreviewedViolations(ctx context.Context, limit int) ([]Violation, error)

	// Admin notifications
	CreateAdminNotification(ctx context.Context, notif AdminNotification) (*AdminNotification, error)
	GetUnreadNotifications(ctx context.Context, limit int) ([]AdminNotification, error)

	// User ban status
	GetUserBanStatus(ctx context.Context, userId string) (*time.Time, error)
	SetUserBan(ctx context.Context, userId string, banUntil time.Time) error
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

// ===== Analysis Logs =====

func (r *repository) CreateAnalysisLog(ctx context.Context, log AnalysisLog) (*AnalysisLog, error) {
	const q = `
		INSERT INTO ai_analysis_logs (
			user_id, request_type, image_url, user_prompt, total_weight,
			response_data, parsed_result, model_used, tokens_prompt, tokens_completion,
			estimated_cost_usd, status, error_message, processing_time_ms
		) VALUES (
			:user_id, :request_type, :image_url, :user_prompt, :total_weight,
			:response_data, :parsed_result, :model_used, :tokens_prompt, :tokens_completion,
			:estimated_cost_usd, :status, :error_message, :processing_time_ms
		) RETURNING id, created_at, updated_at`

	rows, err := r.db.NamedQueryContext(ctx, q, log)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&log.Id, &log.CreatedAt, &log.UpdatedAt); err != nil {
			return nil, err
		}
	}
	return &log, nil
}

func (r *repository) UpdateAnalysisLog(ctx context.Context, log AnalysisLog) error {
	const q = `
		UPDATE ai_analysis_logs SET
			response_data = :response_data,
			parsed_result = :parsed_result,
			tokens_prompt = :tokens_prompt,
			tokens_completion = :tokens_completion,
			estimated_cost_usd = :estimated_cost_usd,
			status = :status,
			error_message = :error_message,
			processing_time_ms = :processing_time_ms,
			updated_at = NOW()
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, q, log)
	return err
}

func (r *repository) GetAnalysisLogById(ctx context.Context, id int64) (*AnalysisLog, error) {
	const q = `SELECT * FROM ai_analysis_logs WHERE id = $1`
	var log AnalysisLog
	if err := r.db.GetContext(ctx, &log, q, id); err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *repository) GetUserAnalysisLogs(ctx context.Context, userId string, limit int) ([]AnalysisLog, error) {
	const q = `
		SELECT * FROM ai_analysis_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	var logs []AnalysisLog
	if err := r.db.SelectContext(ctx, &logs, q, userId, limit); err != nil {
		return nil, err
	}
	return logs, nil
}

// ===== User Limits =====

func (r *repository) GetOrCreateUserLimit(ctx context.Context, userId, requestType string, date time.Time) (*UserLimit, error) {
	// Check if user has active subscription
	hasActiveSubscription, subscriptionTier := r.checkUserSubscription(ctx, userId)

	// Set limit parameters based on subscription
	var limitDate time.Time
	var maxRequests int
	var tier string

	if hasActiveSubscription {
		// Premium users: 10 per day
		limitDate = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		maxRequests = 10
		tier = subscriptionTier
	} else {
		// Free users: 3 per week (Monday-based)
		limitDate = getStartOfWeek(date)
		maxRequests = 3
		tier = "free"
	}

	dateStr := limitDate.Format("2006-01-02")

	// Try to get existing
	existing, err := r.GetUserLimit(ctx, userId, requestType, limitDate)
	if err == nil && existing != nil {
		return existing, nil
	}

	// Create new limit
	const q = `
		INSERT INTO ai_user_limits (user_id, limit_date, request_type, requests_count, max_requests, subscription_tier)
		VALUES ($1, $2, $3, 0, $4, $5)
		ON CONFLICT (user_id, limit_date, request_type) DO UPDATE
		SET max_requests = EXCLUDED.max_requests,
		    subscription_tier = EXCLUDED.subscription_tier,
		    updated_at = NOW()
		RETURNING *`

	var limit UserLimit
	if err := r.db.GetContext(ctx, &limit, q, userId, dateStr, requestType, maxRequests, tier); err != nil {
		return nil, err
	}
	return &limit, nil
}

// getStartOfWeek returns the Monday of the week for the given date
func getStartOfWeek(t time.Time) time.Time {
	// Get the weekday (Sunday = 0, Monday = 1, ...)
	weekday := int(t.Weekday())

	// Calculate days to subtract to get to Monday
	// If Sunday (0), subtract 6 days; if Monday (1), subtract 0 days; etc.
	daysToMonday := (weekday + 6) % 7

	// Subtract days and reset time to midnight
	monday := t.AddDate(0, 0, -daysToMonday)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
}

// checkUserSubscription checks if user has active subscription
func (r *repository) checkUserSubscription(ctx context.Context, userId string) (bool, string) {
	const q = `
		SELECT status, period_end
		FROM subscriptions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 1`

	var status string
	var periodEnd time.Time

	err := r.db.QueryRowContext(ctx, q, userId).Scan(&status, &periodEnd)
	if err != nil {
		// No subscription found
		return false, "free"
	}

	// Check if subscription is active and not expired
	if status == "active" && periodEnd.After(time.Now()) {
		return true, "premium"
	}

	return false, "free"
}

func (r *repository) IncrementUserLimit(ctx context.Context, userId, requestType string, date time.Time) error {
	// Check subscription to determine limit type
	hasActiveSubscription, _ := r.checkUserSubscription(ctx, userId)

	var limitDate time.Time
	if hasActiveSubscription {
		// Premium users: daily limit
		limitDate = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	} else {
		// Free users: weekly limit (Monday-based)
		limitDate = getStartOfWeek(date)
	}

	dateStr := limitDate.Format("2006-01-02")

	const q = `
		UPDATE ai_user_limits
		SET requests_count = requests_count + 1, updated_at = NOW()
		WHERE user_id = $1 AND limit_date = $2 AND request_type = $3`

	_, err := r.db.ExecContext(ctx, q, userId, dateStr, requestType)
	return err
}

func (r *repository) GetUserLimit(ctx context.Context, userId, requestType string, date time.Time) (*UserLimit, error) {
	// Check subscription to determine limit type
	hasActiveSubscription, _ := r.checkUserSubscription(ctx, userId)

	var limitDate time.Time
	if hasActiveSubscription {
		// Premium users: daily limit
		limitDate = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	} else {
		// Free users: weekly limit (Monday-based)
		limitDate = getStartOfWeek(date)
	}

	dateStr := limitDate.Format("2006-01-02")

	const q = `
		SELECT * FROM ai_user_limits
		WHERE user_id = $1 AND limit_date = $2 AND request_type = $3`

	var limit UserLimit
	if err := r.db.GetContext(ctx, &limit, q, userId, dateStr, requestType); err != nil {
		return nil, err
	}
	return &limit, nil
}

// ===== Violations =====

func (r *repository) CreateViolation(ctx context.Context, violation Violation) (*Violation, error) {
	const q = `
		INSERT INTO ai_violations (
			user_id, analysis_log_id, violation_type, violation_reason,
			image_url, user_prompt, action_taken, ban_until
		) VALUES (
			:user_id, :analysis_log_id, :violation_type, :violation_reason,
			:image_url, :user_prompt, :action_taken, :ban_until
		) RETURNING id, created_at, updated_at`

	rows, err := r.db.NamedQueryContext(ctx, q, violation)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&violation.Id, &violation.CreatedAt, &violation.UpdatedAt); err != nil {
			return nil, err
		}
	}
	return &violation, nil
}

func (r *repository) GetUserActiveViolations(ctx context.Context, userId string) ([]Violation, error) {
	const q = `
		SELECT * FROM ai_violations
		WHERE user_id = $1 AND reviewed = false
		ORDER BY created_at DESC`

	var violations []Violation
	if err := r.db.SelectContext(ctx, &violations, q, userId); err != nil {
		return nil, err
	}
	return violations, nil
}

func (r *repository) GetUnreviewedViolations(ctx context.Context, limit int) ([]Violation, error) {
	const q = `
		SELECT * FROM ai_violations
		WHERE reviewed = false
		ORDER BY created_at DESC
		LIMIT $1`

	var violations []Violation
	if err := r.db.SelectContext(ctx, &violations, q, limit); err != nil {
		return nil, err
	}
	return violations, nil
}

// ===== Admin Notifications =====

func (r *repository) CreateAdminNotification(ctx context.Context, notif AdminNotification) (*AdminNotification, error) {
	const q = `
		INSERT INTO admin_notifications (
			notification_type, title, message, severity,
			user_id, related_id, metadata
		) VALUES (
			:notification_type, :title, :message, :severity,
			:user_id, :related_id, :metadata
		) RETURNING id, created_at, updated_at`

	rows, err := r.db.NamedQueryContext(ctx, q, notif)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&notif.Id, &notif.CreatedAt, &notif.UpdatedAt); err != nil {
			return nil, err
		}
	}
	return &notif, nil
}

func (r *repository) GetUnreadNotifications(ctx context.Context, limit int) ([]AdminNotification, error) {
	const q = `
		SELECT * FROM admin_notifications
		WHERE read = false
		ORDER BY severity DESC, created_at DESC
		LIMIT $1`

	var notifs []AdminNotification
	if err := r.db.SelectContext(ctx, &notifs, q, limit); err != nil {
		return nil, err
	}
	return notifs, nil
}

// ===== User Ban Status =====

func (r *repository) GetUserBanStatus(ctx context.Context, userId string) (*time.Time, error) {
	const q = `SELECT ai_banned_until FROM users WHERE id = $1`
	var banUntil *time.Time
	if err := r.db.GetContext(ctx, &banUntil, q, userId); err != nil {
		return nil, err
	}
	return banUntil, nil
}

func (r *repository) SetUserBan(ctx context.Context, userId string, banUntil time.Time) error {
	const q = `UPDATE users SET ai_banned_until = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, q, banUntil, userId)
	return err
}
