package supplement

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jourloy/nutri02/internal/database"
)

type Repository interface {
	// Templates
	GetAllTemplates(ctx context.Context) ([]SupplementTemplate, error)
	GetTemplateByID(ctx context.Context, id int64) (*SupplementTemplate, error)

	// Supplements CRUD
	CreateSupplement(ctx context.Context, userID string, req SupplementCreateRequest) (*Supplement, error)
	GetUserSupplements(ctx context.Context, userID string, activeOnly bool) ([]Supplement, error)
	GetSupplementByID(ctx context.Context, supplementID string, userID string) (*Supplement, error)
	UpdateSupplement(ctx context.Context, supplementID string, userID string, req SupplementCreateRequest) (*Supplement, error)
	DeleteSupplement(ctx context.Context, supplementID string, userID string) error
	GetActiveSupplementsWithSchedules(ctx context.Context, userID string) ([]Supplement, error)

	// Schedules
	CreateSchedules(ctx context.Context, supplementID string, schedules []SupplementScheduleCreate) ([]SupplementSchedule, error)
	GetSchedulesBySupplementID(ctx context.Context, supplementID string) ([]SupplementSchedule, error)
	DeleteSchedulesBySupplementID(ctx context.Context, supplementID string) error

	// Intakes
	CreateIntake(ctx context.Context, intake SupplementIntake) (*SupplementIntake, error)
	GetIntakeByID(ctx context.Context, intakeID string) (*SupplementIntake, error)
	DeleteIntake(ctx context.Context, intakeID string) error
	GetIntakeForToday(ctx context.Context, supplementID string, scheduleID string, date string) (*SupplementIntake, error)
	GetIntakeHistory(ctx context.Context, userID string, params IntakeHistoryParams) ([]SupplementIntake, error)
	GetMissedIntakesFromYesterday(ctx context.Context, userID string, yesterday string) ([]SupplementIntake, error)

	// Notifications
	LogNotification(ctx context.Context, log SupplementNotificationLog) error
	IsNotificationSent(ctx context.Context, scheduleID string, scheduledFor time.Time, notificationType string) (bool, error)

	// Statistics
	GetStatistics(ctx context.Context, userID string) (*SupplementStatistics, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository() Repository {
	return &repository{db: database.Database}
}

// === Templates ===

func (r *repository) GetAllTemplates(ctx context.Context) ([]SupplementTemplate, error) {
	const q = `
	SELECT id, slug, name_ru, name_en, category, icon, description_ru, description_en, sort_order, created_at, updated_at
	FROM supplements_templates
	ORDER BY sort_order ASC, name_en ASC`

	var templates []SupplementTemplate
	if err := r.db.SelectContext(ctx, &templates, q); err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *repository) GetTemplateByID(ctx context.Context, id int64) (*SupplementTemplate, error) {
	const q = `
	SELECT id, slug, name_ru, name_en, category, icon, description_ru, description_en, sort_order, created_at, updated_at
	FROM supplements_templates
	WHERE id = $1`

	var template SupplementTemplate
	if err := r.db.GetContext(ctx, &template, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &template, nil
}

// === Supplements CRUD ===

func (r *repository) CreateSupplement(ctx context.Context, userID string, req SupplementCreateRequest) (*Supplement, error) {
	// Parse start date
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date format: %w", err)
	}

	var endDate *time.Time
	if req.EndDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end_date format: %w", err)
		}
		endDate = &parsed
	}

	const q = `
	INSERT INTO supplements (user_id, template_id, custom_name, start_date, end_date, is_active, notes)
	VALUES ($1, $2, $3, $4, $5, true, $6)
	RETURNING id, user_id, template_id, custom_name, start_date, end_date, is_active, notes, created_at, updated_at, deleted_at`

	var supplement Supplement
	err = r.db.QueryRowxContext(ctx, q, userID, req.TemplateID, req.CustomName, startDate, endDate, req.Notes).StructScan(&supplement)
	if err != nil {
		return nil, err
	}

	return &supplement, nil
}

func (r *repository) GetUserSupplements(ctx context.Context, userID string, activeOnly bool) ([]Supplement, error) {
	q := `
	SELECT s.id, s.user_id, s.template_id, s.custom_name, s.start_date, s.end_date, s.is_active, s.notes,
	       s.created_at, s.updated_at, s.deleted_at,
	       t.id as "template.id", t.slug as "template.slug", t.name_ru as "template.name_ru",
	       t.name_en as "template.name_en", t.category as "template.category", t.icon as "template.icon",
	       t.sort_order as "template.sort_order", t.created_at as "template.created_at", t.updated_at as "template.updated_at"
	FROM supplements s
	LEFT JOIN supplements_templates t ON s.template_id = t.id
	WHERE s.user_id = $1 AND s.deleted_at IS NULL`

	if activeOnly {
		q += ` AND s.is_active = true`
	}

	q += ` ORDER BY s.created_at DESC`

	rows, err := r.db.QueryxContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var supplements []Supplement
	for rows.Next() {
		var s Supplement
		var template SupplementTemplate
		var templateID *int64

		// Scan with template fields
		err := rows.Scan(
			&s.ID, &s.UserID, &s.TemplateID, &s.CustomName, &s.StartDate, &s.EndDate, &s.IsActive, &s.Notes,
			&s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
			&templateID, &template.Slug, &template.NameRu, &template.NameEn, &template.Category, &template.Icon,
			&template.SortOrder, &template.CreatedAt, &template.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		// If template exists, set it
		if templateID != nil {
			template.ID = *templateID
			s.Template = &template
		}

		supplements = append(supplements, s)
	}

	return supplements, nil
}

func (r *repository) GetSupplementByID(ctx context.Context, supplementID string, userID string) (*Supplement, error) {
	const q = `
	SELECT s.id, s.user_id, s.template_id, s.custom_name, s.start_date, s.end_date, s.is_active, s.notes,
	       s.created_at, s.updated_at, s.deleted_at,
	       t.id as "template.id", t.slug as "template.slug", t.name_ru as "template.name_ru",
	       t.name_en as "template.name_en", t.category as "template.category", t.icon as "template.icon",
	       t.sort_order as "template.sort_order", t.created_at as "template.created_at", t.updated_at as "template.updated_at"
	FROM supplements s
	LEFT JOIN supplements_templates t ON s.template_id = t.id
	WHERE s.id = $1 AND s.user_id = $2 AND s.deleted_at IS NULL`

	rows, err := r.db.QueryxContext(ctx, q, supplementID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, sql.ErrNoRows
	}

	var s Supplement
	var template SupplementTemplate
	var templateID *int64

	err = rows.Scan(
		&s.ID, &s.UserID, &s.TemplateID, &s.CustomName, &s.StartDate, &s.EndDate, &s.IsActive, &s.Notes,
		&s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
		&templateID, &template.Slug, &template.NameRu, &template.NameEn, &template.Category, &template.Icon,
		&template.SortOrder, &template.CreatedAt, &template.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if templateID != nil {
		template.ID = *templateID
		s.Template = &template
	}

	return &s, nil
}

func (r *repository) UpdateSupplement(ctx context.Context, supplementID string, userID string, req SupplementCreateRequest) (*Supplement, error) {
	// Parse dates
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date format: %w", err)
	}

	var endDate *time.Time
	if req.EndDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.EndDate)
		if err != nil {
			return nil, fmt.Errorf("invalid end_date format: %w", err)
		}
		endDate = &parsed
	}

	const q = `
	UPDATE supplements
	SET template_id = $1, custom_name = $2, start_date = $3, end_date = $4, notes = $5, updated_at = NOW()
	WHERE id = $6 AND user_id = $7 AND deleted_at IS NULL
	RETURNING id, user_id, template_id, custom_name, start_date, end_date, is_active, notes, created_at, updated_at, deleted_at`

	var supplement Supplement
	err = r.db.QueryRowxContext(ctx, q, req.TemplateID, req.CustomName, startDate, endDate, req.Notes, supplementID, userID).StructScan(&supplement)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("supplement not found")
		}
		return nil, err
	}

	return &supplement, nil
}

func (r *repository) DeleteSupplement(ctx context.Context, supplementID string, userID string) error {
	const q = `
	UPDATE supplements
	SET deleted_at = NOW()
	WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, q, supplementID, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("supplement not found")
	}

	return nil
}

func (r *repository) GetActiveSupplementsWithSchedules(ctx context.Context, userID string) ([]Supplement, error) {
	// Get active supplements
	supplements, err := r.GetUserSupplements(ctx, userID, true)
	if err != nil {
		return nil, err
	}

	// Load schedules for each supplement
	for i := range supplements {
		schedules, err := r.GetSchedulesBySupplementID(ctx, supplements[i].ID)
		if err != nil {
			return nil, err
		}
		supplements[i].Schedules = schedules
	}

	return supplements, nil
}

// === Schedules ===

func (r *repository) CreateSchedules(ctx context.Context, supplementID string, schedules []SupplementScheduleCreate) ([]SupplementSchedule, error) {
	if len(schedules) == 0 {
		return []SupplementSchedule{}, nil
	}

	const q = `
	INSERT INTO supplement_schedules (
		supplement_id, frequency_type, intake_time, days_of_week, interval_days, day_of_month,
		enable_notification, notification_time
	) VALUES (
		:supplement_id, :frequency_type, :intake_time, :days_of_week, :interval_days, :day_of_month,
		:enable_notification, :notification_time
	) RETURNING id, supplement_id, frequency_type, intake_time, days_of_week, interval_days, day_of_month,
		enable_notification, notification_time, created_at, updated_at`

	var result []SupplementSchedule
	for _, schedule := range schedules {
		params := map[string]interface{}{
			"supplement_id":       supplementID,
			"frequency_type":      schedule.FrequencyType,
			"intake_time":         schedule.IntakeTime,
			"days_of_week":        IntSliceToIntArray(schedule.DaysOfWeek),
			"interval_days":       schedule.IntervalDays,
			"day_of_month":        schedule.DayOfMonth,
			"enable_notification": schedule.EnableNotification,
			"notification_time":   schedule.NotificationTime,
		}

		rows, err := r.db.NamedQueryContext(ctx, q, params)
		if err != nil {
			return nil, err
		}

		if rows.Next() {
			var created SupplementSchedule
			if err := rows.StructScan(&created); err != nil {
				rows.Close()
				return nil, err
			}
			result = append(result, created)
		}
		rows.Close()
	}

	return result, nil
}

func (r *repository) GetSchedulesBySupplementID(ctx context.Context, supplementID string) ([]SupplementSchedule, error) {
	const q = `
	SELECT id, supplement_id, frequency_type, intake_time, days_of_week, interval_days, day_of_month,
	       enable_notification, notification_time, created_at, updated_at
	FROM supplement_schedules
	WHERE supplement_id = $1
	ORDER BY created_at ASC`

	var schedules []SupplementSchedule
	if err := r.db.SelectContext(ctx, &schedules, q, supplementID); err != nil {
		return nil, err
	}
	return schedules, nil
}

func (r *repository) DeleteSchedulesBySupplementID(ctx context.Context, supplementID string) error {
	const q = `DELETE FROM supplement_schedules WHERE supplement_id = $1`
	_, err := r.db.ExecContext(ctx, q, supplementID)
	return err
}

// === Intakes ===

func (r *repository) CreateIntake(ctx context.Context, intake SupplementIntake) (*SupplementIntake, error) {
	const q = `
	INSERT INTO supplement_intakes (
		supplement_id, schedule_id, user_id, scheduled_at, taken_at, is_on_time, is_missed, source
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING id, supplement_id, schedule_id, user_id, scheduled_at, taken_at, is_on_time, is_missed, source, created_at`

	var created SupplementIntake
	err := r.db.QueryRowxContext(ctx, q,
		intake.SupplementID, intake.ScheduleID, intake.UserID, intake.ScheduledAt,
		intake.TakenAt, intake.IsOnTime, intake.IsMissed, intake.Source,
	).StructScan(&created)
	if err != nil {
		return nil, err
	}

	return &created, nil
}

func (r *repository) GetIntakeByID(ctx context.Context, intakeID string) (*SupplementIntake, error) {
	const q = `
	SELECT id, supplement_id, schedule_id, user_id, scheduled_at, taken_at, is_on_time, is_missed, source, created_at
	FROM supplement_intakes
	WHERE id = $1`

	var intake SupplementIntake
	err := r.db.GetContext(ctx, &intake, q, intakeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("intake not found")
		}
		return nil, err
	}
	return &intake, nil
}

func (r *repository) DeleteIntake(ctx context.Context, intakeID string) error {
	const q = `DELETE FROM supplement_intakes WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, intakeID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("intake not found")
	}

	return nil
}

func (r *repository) GetIntakeForToday(ctx context.Context, supplementID string, scheduleID string, date string) (*SupplementIntake, error) {
	const q = `
	SELECT id, supplement_id, schedule_id, user_id, scheduled_at, taken_at, is_on_time, is_missed, source, created_at
	FROM supplement_intakes
	WHERE supplement_id = $1 AND schedule_id = $2 AND DATE(scheduled_at) = $3
	ORDER BY created_at DESC
	LIMIT 1`

	var intake SupplementIntake
	err := r.db.GetContext(ctx, &intake, q, supplementID, scheduleID, date)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &intake, nil
}

func (r *repository) GetIntakeHistory(ctx context.Context, userID string, params IntakeHistoryParams) ([]SupplementIntake, error) {
	q := `
	SELECT i.id, i.supplement_id, i.schedule_id, i.user_id, i.scheduled_at, i.taken_at,
	       i.is_on_time, i.is_missed, i.source, i.created_at
	FROM supplement_intakes i
	WHERE i.user_id = $1`

	args := []interface{}{userID}
	argCount := 1

	if params.Date != nil {
		argCount++
		q += fmt.Sprintf(` AND DATE(i.scheduled_at) = $%d`, argCount)
		args = append(args, *params.Date)
	}

	if params.SupplementID != nil {
		argCount++
		q += fmt.Sprintf(` AND i.supplement_id = $%d`, argCount)
		args = append(args, *params.SupplementID)
	}

	q += ` ORDER BY i.scheduled_at DESC`

	if params.Limit > 0 {
		argCount++
		q += fmt.Sprintf(` LIMIT $%d`, argCount)
		args = append(args, params.Limit)
	}

	if params.Offset > 0 {
		argCount++
		q += fmt.Sprintf(` OFFSET $%d`, argCount)
		args = append(args, params.Offset)
	}

	var intakes []SupplementIntake
	if err := r.db.SelectContext(ctx, &intakes, q, args...); err != nil {
		return nil, err
	}

	return intakes, nil
}

func (r *repository) GetMissedIntakesFromYesterday(ctx context.Context, userID string, yesterday string) ([]SupplementIntake, error) {
	const q = `
	SELECT i.id, i.supplement_id, i.schedule_id, i.user_id, i.scheduled_at, i.taken_at,
	       i.is_on_time, i.is_missed, i.source, i.created_at
	FROM supplement_intakes i
	WHERE i.user_id = $1 AND DATE(i.scheduled_at) = $2 AND i.is_missed = true
	ORDER BY i.scheduled_at ASC`

	var intakes []SupplementIntake
	if err := r.db.SelectContext(ctx, &intakes, q, userID, yesterday); err != nil {
		return nil, err
	}

	return intakes, nil
}

// === Notifications ===

func (r *repository) LogNotification(ctx context.Context, log SupplementNotificationLog) error {
	const q = `
	INSERT INTO supplement_notification_log (
		supplement_id, schedule_id, user_id, telegram_id, scheduled_for, notification_type
	) VALUES ($1, $2, $3, $4, $5, $6)
	ON CONFLICT (schedule_id, scheduled_for, notification_type) DO NOTHING`

	_, err := r.db.ExecContext(ctx, q, log.SupplementID, log.ScheduleID, log.UserID, log.TelegramID, log.ScheduledFor, log.NotificationType)
	return err
}

func (r *repository) IsNotificationSent(ctx context.Context, scheduleID string, scheduledFor time.Time, notificationType string) (bool, error) {
	const q = `
	SELECT EXISTS(
		SELECT 1 FROM supplement_notification_log
		WHERE schedule_id = $1 AND scheduled_for = $2 AND notification_type = $3
	)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, q, scheduleID, scheduledFor, notificationType)
	return exists, err
}

// === Statistics ===

func (r *repository) GetStatistics(ctx context.Context, userID string) (*SupplementStatistics, error) {
	// Count supplements
	const qSupplements = `
	SELECT
		COUNT(*) as total,
		COUNT(*) FILTER (WHERE is_active = true) as active
	FROM supplements
	WHERE user_id = $1 AND deleted_at IS NULL`

	var stats SupplementStatistics
	err := r.db.QueryRowContext(ctx, qSupplements, userID).Scan(&stats.TotalSupplements, &stats.ActiveSupplements)
	if err != nil {
		return nil, err
	}

	// Count intakes
	const qIntakes = `
	SELECT
		COUNT(*) as total,
		COUNT(*) FILTER (WHERE is_missed = true) as missed
	FROM supplement_intakes
	WHERE user_id = $1`

	err = r.db.QueryRowContext(ctx, qIntakes, userID).Scan(&stats.TotalIntakes, &stats.MissedIntakes)
	if err != nil {
		return nil, err
	}

	// Calculate miss rate
	if stats.TotalIntakes > 0 {
		stats.MissRate = float64(stats.MissedIntakes) / float64(stats.TotalIntakes) * 100
	}

	// Calculate current streak (consecutive days without misses)
	// This is simplified - a full implementation would check actual scheduled days
	const qStreak = `
	WITH daily_status AS (
		SELECT
			DATE(scheduled_at) as day,
			bool_or(is_missed) as has_miss
		FROM supplement_intakes
		WHERE user_id = $1
		GROUP BY DATE(scheduled_at)
		ORDER BY DATE(scheduled_at) DESC
	)
	SELECT COUNT(*)
	FROM daily_status
	WHERE NOT has_miss`

	err = r.db.GetContext(ctx, &stats.CurrentStreak, qStreak, userID)
	if err != nil {
		stats.CurrentStreak = 0
	}

	// For MVP, longest streak = current streak
	// A full implementation would need more complex logic
	stats.LongestStreak = stats.CurrentStreak

	return &stats, nil
}
